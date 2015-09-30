var _ = require('lodash')
var merge = require('merge-stream')
var gulp = require('gulp')
var gutil = require('gulp-util')
var gfile = require('gulp-file')
var gzip = require('gulp-gzip')
var gtemplate = require('gulp-template')
var less = require('gulp-less')
var autoprefixer = require('gulp-autoprefixer')
var uglify = require('gulp-uglify')
var sourcemaps = require('gulp-sourcemaps')
var source = require('vinyl-source-stream')
var buffer = require('vinyl-buffer')
var watchify = require('watchify')
var browserify = require('browserify')
var envify = require('envify/custom')
var react = require('gulp-react')
var jshint = require('gulp-jshint')
var serve = require('gulp-serve')
var fs = require('fs')
var path = require('path')
var exec = require('child_process').exec

require('node-jsx').install()

var watching = false
var heimDest = './build/heim'
var embedDest = './build/embed'
var emailDest = './build/email'

var heimOptions = {
  HEIM_ORIGIN: process.env.HEIM_ORIGIN,
  HEIM_PREFIX: process.env.HEIM_PREFIX || '',
  EMBED_ORIGIN: process.env.EMBED_ORIGIN,
  NODE_ENV: process.env.NODE_ENV,
}

// via https://github.com/tblobaum/git-rev
function shell(cmd, cb) {
  exec(cmd, { cwd: __dirname }, function(err, stdout) {
    if (err) {
      throw err
    }
    cb(stdout.trim())
  })
}

// FIXME: replace with a more robust js loader
function reload(moduleName) {
  delete require.cache[require.resolve(moduleName)]
  return require(moduleName)
}

function handleError(title) {
  return function(err) {
    gutil.log(gutil.colors.red(title + ':'), err.message)
    if (watching) {
      this.emit('end')
    } else {
      process.exit(1)
    }
  }
}

function heimBundler(args) {
  return browserify('./lib/client.js', args)
    .transform(envify(heimOptions))
}

function embedBundler(args) {
  return browserify('./lib/embed.js', args)
    .transform(envify({
      HEIM_ORIGIN: process.env.HEIM_ORIGIN,
    }))
}

gulp.task('heim-git-commit', function() {
  shell('git rev-parse HEAD', function(gitRev) {
    heimOptions.HEIM_GIT_COMMIT = gitRev
  })
})

gulp.task('heim-js', ['heim-git-commit', 'heim-less'], function() {
  return heimBundler({debug: true})
    // share some libraries with the global namespace
    // doing this here because these exposes trip up watchify atm
    .require('lodash', {expose: 'lodash'})
    .require('react', {expose: 'react'})
    .require('reflux', {expose: 'reflux'})
    .require('immutable', {expose: 'immutable'})
    .require('moment', {expose: 'moment'})
    .require('querystring', {expose: 'querystring'})
    .bundle()
    .pipe(source('main.js'))
    .pipe(buffer())
    .pipe(sourcemaps.init({loadMaps: true}))
      .pipe(process.env.NODE_ENV == 'production' ? uglify() : gutil.noop())
    .pipe(sourcemaps.write('./', {includeContent: true}))
    .on('error', handleError('heim browserify error'))
    .pipe(gulp.dest(heimDest))
    .pipe(gzip())
    .pipe(gulp.dest(heimDest))
})

gulp.task('embed-js', function() {
  return embedBundler({debug: true})
    .bundle()
    .pipe(source('embed.js'))
    .pipe(buffer())
    .pipe(sourcemaps.init({loadMaps: true}))
      .pipe(process.env.NODE_ENV == 'production' ? uglify() : gutil.noop())
    .pipe(sourcemaps.write('./', {includeContent: true}))
    .on('error', handleError('embed browserify error'))
    .pipe(gulp.dest(embedDest))
    .pipe(gzip())
    .pipe(gulp.dest(embedDest))
})

gulp.task('raven-js', ['heim-git-commit', 'heim-js'], function() {
  shell('md5sum build/main.js | cut -d " " -f 1', function(releaseHash) {
    return browserify('./lib/raven.js')
      .transform(envify(_.extend({
        SENTRY_ENDPOINT: process.env.SENTRY_ENDPOINT,
        HEIM_RELEASE: releaseHash,
      }, heimOptions)))
      .bundle()
      .pipe(source('raven.js'))
      .pipe(buffer())
      .pipe(process.env.NODE_ENV == 'production' ? uglify() : gutil.noop())
      .on('error', handleError('raven browserify error'))
      .pipe(gulp.dest(heimDest))
      .pipe(gzip())
      .pipe(gulp.dest(heimDest))
  })
})

gulp.task('heim-less', function() {
  return gulp.src(['./lib/main.less', './lib/crashed.less', './lib/od.less', './site/*.less'])
    .pipe(less({compress: true}))
    .on('error', handleError('LESS error'))
    .pipe(autoprefixer({cascade: false}))
    .on('error', handleError('autoprefixer error'))
    .pipe(gulp.dest(heimDest))
    .pipe(gzip())
    .pipe(gulp.dest(heimDest))
})

gulp.task('emoji-less', function() {
  var emoji = require('./lib/emoji')
  var twemojiPath = path.dirname(require.resolve('twemoji')) + '/svg/'
  var leadingZeroes = /^0*/
  var source = _.map(emoji.codes, function(code) {
    if (!code) {
      return
    }
    var twemojiName = code.replace(leadingZeroes, '')
    var emojiPath = './res/emoji/' + twemojiName + '.svg'
    if (!fs.existsSync(emojiPath)) {
      emojiPath = twemojiPath + twemojiName + '.svg'
    }
    return '.emoji-' + code + ' { background-image: data-uri("' + emojiPath + '") }'
  }).join('\n')
  return gfile('emoji.less', source, {src: true})
    .pipe(less({compress: true}))
    .pipe(gulp.dest(heimDest))
    .pipe(gzip())
    .pipe(gulp.dest(heimDest))
})

gulp.task('heim-static', function() {
  return gulp.src('./static/**/*')
    .pipe(gulp.dest(heimDest))
})

gulp.task('embed-static', function() {
  return gulp.src('./static/robots.txt')
    .pipe(gulp.dest(embedDest))
})

gulp.task('heim-html', ['heim-git-commit'], function() {
  return gulp.src(['./lib/room.html'])
    .pipe(gtemplate(heimOptions))
    .pipe(gulp.dest(heimDest))
})

gulp.task('embed-html', function() {
  return gulp.src(['./lib/embed.html'])
    .pipe(gulp.dest(embedDest))
})

gulp.task('site-templates', function() {
  var page = reload('./site/page.js')
  var pages = ['home', 'about/values', 'about/conduct']

  return merge(_.map(pages, function(name) {
    var html = page.render(reload('./site/' + name))
    return gfile(name + '.html', html, {src: true})
  }))
    .pipe(gulp.dest(heimDest))
})

gulp.task('email-templates', function() {
  var email = reload('./emails/email.js')
  var emails = ['welcome', 'room-invitation', 'room-invitation-welcome', 'password-changed', 'password-reset']

  var htmls = merge(_.map(emails, function(name) {
    var html = email.renderEmail(reload('./emails/' + name))
    return gfile(name + '.html', html, {src: true})
  }))

  var txtCommon = reload('./emails/common-txt.js')
  var txts = merge(_.map(emails, function(name) {
    return gulp.src('./emails/' + name + '.txt')
      .pipe(gtemplate(txtCommon))
  }))

  return merge(htmls, txts)
    .pipe(gulp.dest(emailDest))
})

gulp.task('email-hdrs', function() {
  return gulp.src('./emails/*.hdr').pipe(gulp.dest(emailDest))
})

gulp.task('email-static', function() {
  return gulp.src('./emails/static/*.png').pipe(gulp.dest(emailDest+'/static'))
})

gulp.task('lint', function() {
  return gulp.src(['./lib/**/*.js', './emails/*.js', './test/**/*.js', './gulpfile.js'])
    .pipe(react())
    .pipe(jshint())
    .pipe(jshint.reporter('default'))
    .pipe(jshint.reporter('fail'))
})

function watchifyTask(name, bundler, outFile, dest) {
  gulp.task(name, ['build-statics'], function() {
    // via https://github.com/gulpjs/gulp/blob/master/docs/recipes/fast-browserify-builds-with-watchify.md
    bundler = watchify(bundler(watchify.args))
    bundler.on('log', gutil.log.bind(gutil, gutil.colors.green('JS (' + name + ')')))
    bundler.on('update', rebundle)

    function rebundle() {
      return bundler.bundle()
        .on('error', handleError('JS (' + name + ') error'))
        .pipe(source(outFile))
        .pipe(gulp.dest(dest))
        .pipe(gzip())
        .pipe(gulp.dest(dest))
    }

    return rebundle()
  })
}

watchifyTask('heim-watchify', heimBundler, 'main.js', heimDest)
watchifyTask('embed-watchify', embedBundler, 'embed.js', embedDest)

gulp.task('build-emails', ['email-templates', 'email-hdrs', 'email-static'])
gulp.task('build-statics', ['raven-js', 'heim-less', 'emoji-less', 'heim-static', 'embed-static', 'heim-html', 'embed-html', 'site-templates'])
gulp.task('build-browserify', ['heim-js', 'embed-js'])

gulp.task('watch', function() {
  watching = true
  gulp.watch('./lib/**/*.less', ['heim-less'])
  gulp.watch('./res/**/*', ['heim-less', 'emoji-less'])
  gulp.watch('./site/**/*.less', ['heim-less'])
  gulp.watch('./lib/**/*.html', ['heim-html', 'embed-html'])
  gulp.watch('./static/**/*', ['heim-static', 'embed-static'])
  gulp.watch('./site/**/*', ['site-templates'])
  gulp.watch('./emails/*', ['email-templates'])
  gulp.watch('./emails/static/*', ['email-static'])
})

gulp.task('build', ['build-statics', 'build-browserify', 'build-emails'])
gulp.task('default', ['build-statics', 'build-emails', 'watch', 'heim-watchify', 'embed-watchify'])

gulp.task('serve-heim', serve({
  port: 8080,
  root: heimDest,
  middleware: function(req, res, next) {
    req.url = req.url.replace(/^\/static|^\/room\/\w+/, '/')
    next()
  },
}))

gulp.task('serve-embed', serve({
  port: 8081,
  root: embedDest,
  middleware: function(req, res, next) {
    req.url = req.url.replace(/^\/\?\S+/, '/embed.html')
    next()
  },
}))

gulp.task('develop', ['serve-heim', 'serve-embed', 'default'])
