# Homebrew Formula for CodeContext
class Codecontext < Formula
  desc "Intelligent context maps for AI-powered development tools"
  homepage "https://github.com/nmakod/codecontext"
  url "https://github.com/nmakod/codecontext/archive/v2.0.0.tar.gz"
  sha256 "ffba3ccfe55ef4012000d6d14ee2ed7fb69c3179a15d4f7ec144277469a60193"
  license "MIT"
  head "https://github.com/nmakod/codecontext.git", branch: "main"

  depends_on "go" => :build

  def install
    # Set version information during build
    ldflags = %W[
      -s -w
      -X main.version=#{version}
      -X main.buildDate=#{time.iso8601}
      -X main.gitCommit=#{Utils.git_head}
    ]

    system "go", "build", *std_go_args(ldflags: ldflags), "./cmd/codecontext"
  end

  test do
    # Test that the binary runs and shows help
    assert_match "CodeContext", shell_output("#{bin}/codecontext --help")
    
    # Test version command
    assert_match version.to_s, shell_output("#{bin}/codecontext --version")
    
    # Test basic functionality with a simple project
    (testpath/"test.ts").write <<~EOS
      export function hello(name: string): string {
        return `Hello, ${name}!`;
      }
    EOS
    
    (testpath/".codecontext").mkdir
    (testpath/".codecontext/config.yaml").write <<~EOS
      project:
        name: "test"
        path: "."
      parser:
        languages: ["typescript"]
      output:
        format: "markdown"
    EOS
    
    # Test that codecontext can analyze the test file
    system bin/"codecontext", "init", "--force"
    assert_predicate testpath/".codecontext/config.yaml", :exist?
    
    system bin/"codecontext", "generate", "--output", "test-output.md"
    assert_predicate testpath/"test-output.md", :exist?
    assert_match "hello", File.read(testpath/"test-output.md")
  end
end