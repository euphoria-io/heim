def _gulp_task(ctx):
    outdir = ctx.actions.declare_file('gulp-' + ctx.attr.name)
    ctx.actions.run_shell(
        outputs = [outdir],
        command = " && ".join([
            "mkdir work",
            "cp -a $(pwd)/{gulp}.runfiles/__main__/client/* work".format(gulp=ctx.executable._gulp.path),
            "ln -sf $(pwd)/{gulp} work/gulp".format(gulp=ctx.executable._gulp.path),
            "mkdir {outdir}".format(outdir=outdir.path),
            "ln -sf $(pwd)/{outdir} work/build".format(outdir=outdir.path),
            "(cd work; ./gulp {name})".format(name=ctx.attr.name),
        ]),
        inputs = [ctx.executable._gulp],
    )
    inputs = []
    for d in ctx.attr.outs:
        f = ctx.actions.declare_file(ctx.attr.renames.get(d, d))
        inputs.append(f)
        ctx.actions.run_shell(
            outputs = [f],
            command = "mkdir -p $1; cp -a $1 $2",
            arguments = [outdir.path + "/" + d, f.path],
            inputs = [outdir],
        )
    ctx.actions.run_shell(
        outputs = [ctx.outputs.zip],
        command = " && ".join([
            "dest=$(pwd)/{out}".format(out=ctx.outputs.zip.path),
            "cd bazel-out/local-fastbuild/bin/client",
            "zip -r $dest {dirs}".format(dirs=' '.join(ctx.attr.outs)),
        ]),
        inputs = inputs,
    )

gulp_task = rule(
    implementation = _gulp_task,
    attrs = {
        "outs": attr.string_list(mandatory = True, allow_empty = False),
        "renames": attr.string_dict(mandatory = True, allow_empty = True),
        "_gulp": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//tools/js:gulp"),
        ),
    },
    outputs = {
        "zip": "%{name}.zip",
    },
)
