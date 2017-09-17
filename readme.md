
# [modernizer](https://github.com/cdelorme/modernizer)

A single exposed function that accepts the current version and the CDN to check for a new version making upgrading the current application dead simple.


## sales pitch

**As usual, my goal was making code that can be understood at-a-glance.**

This code provides a single function call, accepting a version and CDN prefix.  Within that one function it checks a version, uses a standard checksum file and matches new binaries by OS and architecture, handling its own replacement, and restarting the process.

The only other option I found was [`go-update`](https://github.com/inconshreveable/go-update), and [`go-selfupdate`](https://github.com/sanbornm/go-selfupdate) which depends on the former.

It offers several key benefits:

1. It utilizes the new built-in [os.Executable()](https://golang.org/pkg/os/#Executable) from go1.8, allowing us to drop the `osext` dependency for identifying the true executable path (_that's nearly 200 lines of code and a transitive dependency_).
2. It willfully excludes the complexity of binary patching.
3. It automatically relaunches the application, properly handing off pipes (_and even the process identifier on unix/linux platforms_).
4. It has zero third-party dependencies.
5. It is under 200 lines of code; under 500 lines of code counting all unit tests.


### patching

From an efficiency perspective applying patches using a binary delta is appealing versus a full binary replacement, but I would disagree with the perceived benefits and can explain several reasons why.

There is an implicit risk when applying a patch to the existing binary.  If the reason for choosing to use a binary delta was because the binary itself is very large, then it is unlikely the patch can be reliably applied in-memory, thus we are left with creating a copy of the existing binary on-disk.  _This eliminates any assumption of saved disk space._

Creating a patch file is computationally expensive.  Each patch file is also built against two known versions of the binary, which increases then cost linearly based on the number of supported platforms.  _The benefits go provides with its speedy compiler are sacrificed when you need to spend several minutes generating a patch file._

Depending on your strategy the cost either grows exponentially to support a delta for jumping across multiple versions to the current state, or increases in complexity when applying updates due to downloading, validating, and patching each version change chronologically.

Adding binary replacement as a backup behavior for when binary patching is not feasible increases the complexity of your code and the size of your binary.

Finally, the average binary in go is around 10MB, and even some of the most popular and largest applications out there are barely 40MB.  Even with average network speeds the difference in download time for a full binary versus a binary patch plus applying that patch makes is minor.

**In conclusion, binary patching is only valid for a limited number of use-cases; it only sanely applies to single-step-update scenarios, and only when the size of the binary can be measured against your network bandwidth as "significantly large" and any delta produced would be "significantly smaller".**


## testing

You can verify the test cases by running:

	go test -v -cover

_I chose to omit some of the more obvious behaviors, namely error handling for edge-cases and the restart logic that is platform specific, which leaves us at around 80% coverage._


## usage

To use this package, simply `import github.com/cdelorme/modernizer` and run `modernizer.Check(version, cdn)` anywhere you want in your code.

**You get to choose when to run the code, where to get the supplied version and CDN, and how you want to handle failed updates.**  However, when the update is successful the process will be handed over (_and on windows the process will `exit 0` and the new process that inherited its pipes will have a new process identifier_).  Take caution since running the update asynchronously may disrupt other operations.

The recommended implementation is to run it at-launch synchronously and to log the error but continue execution:

	func main() {
		if err := modernizer.Check(version, versionCdn); err != nil {
			// log error, but continue to execute
		}
	}

For open source software, you can leverage github's release system and use tags to identify your binaries.  For example, you could use the `stable` tag in a fashion idiomatic to go, or you could try for semantic versions using tags such as `1.x` or `1.N.x` to automatically upgrade at smaller subsets.

For private software, you'll have to manage a CDN such as AWS S3 with CloudFront, setting up appropriate security groups to limit access, _but this solution does not include support for additional authentication at download time._

Another recommended solution is to use `-ldflags` at build time to set the version and CDN, which decouples them from the code itself.  _My preference would be to use the VCS hash to version your builds, as it can track down an exact point in time in your code and tends to be more reliable than semantic versions:_

	go build -ldflags "-X main.version=$(git rev-parse HEAD) -X main.versionCdn=https://github.com/cdelorme/modernizer/archive/stable"


### expectations

The CDN is expected to have the following files:

- `version`
- `sha256sums`

The version file may contain any string value, and it will be compared to the current application version to determine whether to replace the current binary.  _As a result, this can be used to apply both updates and downgrades or rollbacks, and you may use semantic versions, git hashes, or even random code words at your discretion._

The sha256sums should be the first column followed by a space then the path name to the binary on the CDN.  The binary path should include the OS and Architecture; _all of these are considered valid_:

	6bb2390d695a2d3675154b1d2aa3b41c6a41840571dbce400caa3cb8938533b3 windows-amd64-modernizer.exe
	e4b2985d29c30090cc591deeb7734454198f289aae5e22b1290c950dbd7b64bf linux/amd64/modernizer
	b46dfc509f9e34bc9a1d7ab672a2f619aceb02edbb2b8835cd079db83d4325fc darwin_amd64 modernizer

The hash is used to validate the download.

There is no restriction preventing separation of platforms.  _Thus you could separately manage versions for each platform by changing the CDN or CDN prefix used, and include only one record in the `sha256sums` file._


# references

- [graceful server restart with go](https://blog.scalingo.com/2014/12/19/graceful-server-restart-with-go.html)
- [how to execute a simple windows dos command in golang](http://stackoverflow.com/questions/13008255/how-to-execute-a-simple-windows-dos-command-in-golang)
- [pass stdin to child process after killing parent](http://stackoverflow.com/questions/32600547/pass-stdin-to-child-process-after-killing-parent)
- [shrink your go binaries with this one weird trick](https://blog.filippo.io/shrink-your-go-binaries-with-this-one-weird-trick/)
