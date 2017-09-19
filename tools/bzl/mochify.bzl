def _mochify_test(ctx):
    ctx.actions.run_shell(
        inputs = [ctx.executable._mochify],
        outputs = [ctx.outputs.executable],
        command = " && ".join([
            "root=$(pwd)",
            "cd {mochify}.runfiles/__main__/client".format(
                mochify = ctx.executable._mochify.path,
            ),
            #"pwd > /tmp/t && sleep 1000",
            "PATH=node_modules/.bin:$PATH $root/{mochify}".format(
                mochify = ctx.executable._mochify.path,
            ),
            "touch $root/" + ctx.outputs.executable.path,
        ]),
    )

mochify_test = rule(
    implementation = _mochify_test,
    attrs = {
        "_mochify": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//tools/js:mochify"),
        ),
    },
    test = True,
)
