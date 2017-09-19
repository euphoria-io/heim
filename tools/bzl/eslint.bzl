def _eslint_test(ctx):
    ctx.actions.run_shell(
        inputs = [ctx.executable._eslint],
        outputs = [ctx.outputs.executable],
        command = " && ".join([
            "root=$(pwd)",
            #"pwd > /tmp/t && sleep 1000",
            "cd {eslint}.runfiles/__main__/client".format(
                eslint = ctx.executable._eslint.path,
            ),
            "$root/{eslint} .".format(
                eslint = ctx.executable._eslint.path,
            ),
            "touch $root/" + ctx.outputs.executable.path,
        ]),
    )

eslint_test = rule(
    implementation = _eslint_test,
    attrs = {
        "_eslint": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//tools/js:eslint"),
        ),
    },
    test = True,
)
