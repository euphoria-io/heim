load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

git_repository(
    name = "build_bazel_rules_nodejs",
    remote = "https://github.com/bazelbuild/rules_nodejs.git",
    tag = "0.10.0",
)

load("@build_bazel_rules_nodejs//:defs.bzl", "node_repositories", "npm_install")

node_repositories(
    package_json = ["//client:package.json"],
    node_version = "8.9.1",
)

npm_install(
    name = "client",
    package_json = "//client:package.json",
)

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://github.com/bazelbuild/rules_go/releases/download/0.13.0/rules_go-0.13.0.tar.gz"],
    sha256 = "ba79c532ac400cefd1859cbc8a9829346aa69e3b99482cd5a54432092cbc3933",
)

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.13.0/bazel-gazelle-0.13.0.tar.gz"],
    sha256 = "bc653d3e058964a5a26dcad02b6c72d7d63e6bb88d94704990b908a1445b8758",
)

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_rules_dependencies",
    "go_register_toolchains",
)

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()
