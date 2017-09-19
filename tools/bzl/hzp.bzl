def _hzp_binary(ctx):
    # unzip all srcs to src_files
    binfile = ctx.actions.declare_file(ctx.attr.name)
    ctx.actions.run_shell(
        outputs = [binfile],
        inputs = [ctx.executable.bin],
        command = "install -D $1 $2",
        arguments = [ctx.executable.bin.path, binfile.path],
    )

    tlds = []
    for target in ctx.attr.srcs:
        for zipfile in target.files:
            f = ctx.actions.declare_file('tld-' + zipfile.basename.replace('.zip', ''))
            tlds.append(f)
            ctx.actions.run_shell(
                outputs = [f],
                command = "unzip {zip} -d {dest}".format(
                    zip = zipfile.path,
                    dest = f.path,
                ),
                inputs = target.files,
            )

    # symlink all srcs in staging dir
    stage = ctx.actions.declare_file('stage')
    ctx.actions.run_shell(
        outputs = [stage],
        command = " && ".join([
                "mkdir " + stage.path,
            ] + [
            "cp -a $(pwd)/{dir}/* {stage}".format(
                dir = f.path,
                stage = stage.path,
            ) for f in tlds]),
        inputs = tlds,
    )

    """
    # debug pause
    pause = ctx.actions.declare_file('pause')
    ctx.actions.run_shell(
        outputs = [pause],
        command = "pwd > /tmp/t && echo sleeping && sleep 100000 && touch " + pause.path,
        inputs = [ctx.executable._heimlich, binfile, stage],
    )
    """

    # build hzp in staging dir
    ctx.actions.run_shell(
        outputs = [ctx.outputs.hzp],
        inputs = [ctx.executable._heimlich, binfile, stage],
        command = " && ".join([
            "root=$(pwd)",
            "mkdir h",
            "cd h",
            "cp -a $root/{stage}/* .".format(
                stage = stage.path,
            ),
            "find . -type f | xargs $root/{heimlich} $root/{binfile}".format(
                binfile = binfile.path,
                heimlich = ctx.executable._heimlich.path,
            ),
        ]),
    )

    """
    zips = []
    for target in ctx.attr.srcs:
        zips.extend(target.files.to_list())
    ctx.actions.run_shell(
        outputs = [stage],
        command = " && ".join([
            "ln -s {zips} stage".format(zips = " ".join(f.path for f in zips)),
        ]),
        inputs = ctx.attr.srcs,
    )
    binfile = ctx.actions.declare_file(ctx.attr.name)
    ctx.actions.run_shell(
        outputs = [f],
        inputs = [ctx.executable.bin],
        command = "install -D $1 $2",
        arguments = [ctx.executable.bin.path, f.path],
    )
    #inputs = [f]
    #for target in ctx.attr.srcs:
        #root = target.label.name
        #inputs.extend(target.files.to_list())
    ctx.actions.run(
        outputs = [ctx.outputs.hzp],
        #inputs = inputs,
        inputs = [f, stage],
        command = 
        executable = ctx.executable._heimlich,
        arguments = [f.path for f in inputs],
    )
    """

hzp_binary = rule(
    implementation = _hzp_binary,
    attrs = {
        "bin": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            mandatory = True),
        "srcs": attr.label_list(allow_files = True),
        "_heimlich": attr.label(
            executable = True,
            cfg = "host",
            allow_files = True,
            default = Label("//heimlich"),
        ),
    },
    outputs = {"hzp": "%{name}.hzp"},
)
