def _install_etcd(ctx):
    ver = ctx.attr.version
    url = "https://storage.googleapis.com/etcd/{v}/etcd-{v}-linux-amd64.tar.gz"
    ctx.download_and_extract(
        url = url.format(v=ctx.attr.version),
        output = "etcd",
        stripPrefix = "etcd-{v}-linux-amd64".format(v=ctx.attr.version),
    )
    ctx.file("BUILD", """
package(default_visibility = ["//visibility:public"])

filegroup(name="etcd", srcs = ["etcd/etcd", "etcd/etcdctl"])
""")

install_etcd = repository_rule(
    implementation = _install_etcd,
    attrs = {
      "version": attr.string(default="v3.2.7"),
    },
)
