// string.js (a dep) clashes with core-js string polyfill, so require first
import 'markdown-it-anchor'
import 'babel-polyfill'

import _ from 'lodash'
import merge from 'merge-stream'
import gulp from 'gulp'
import gutil from 'gulp-util'
import gfile from 'gulp-file'
import gzip from 'gulp-gzip'
import gtemplate from 'gulp-template'
import less from 'gulp-less'
import autoprefixer from 'gulp-autoprefixer'
import uglify from 'gulp-uglify'
import sourcemaps from 'gulp-sourcemaps'
import source from 'vinyl-source-stream'
import buffer from 'vinyl-buffer'
import watchify from 'watchify'
import browserify from 'browserify'
import envify from 'envify/custom'
import serve from 'gulp-serve'
import fs from 'fs'
import path from 'path'
import { exec } from 'child_process'

let watching = false
const heimDest = './build/heim'
const embedDest = './build/embed'
const emailDest = './build/email'

const heimOptions = {
  HEIM_ORIGIN: process.env.HEIM_ORIGIN,
  HEIM_PREFIX: process.env.HEIM_PREFIX || '',
  EMBED_ORIGIN: process.env.EMBED_ORIGIN,
  NODE_ENV: process.env.NODE_ENV,
}

// via https://github.com/tblobaum/git-rev
function shell(cmd, cb) {
  exec(cmd, { cwd: __dirname }, function onExecResult(err, stdout) {
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
  return function handler(err) {
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

gulp.task('heim-git-commit', done => {
  shell('git rev-parse HEAD', gitRev => {
    process.env.HEIM_GIT_COMMIT = heimOptions.HEIM_GIT_COMMIT = gitRev
    done()
  })
})

gulp.task('heim-js', ['heim-git-commit', 'heim-less'], () => {
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
      .pipe(process.env.NODE_ENV === 'production' ? uglify() : gutil.noop())
    .pipe(sourcemaps.write('./', {includeContent: true}))
    .on('error', handleError('heim browserify error'))
    .pipe(gulp.dest(heimDest))
    .pipe(gzip())
    .pipe(gulp.dest(heimDest))
})

gulp.task('fast-touch-js', () => {
  return gulp.src('./site/lib/fast-touch.js')
    .pipe(process.env.NODE_ENV === 'production' ? uglify() : gutil.noop())
    .on('error', handleError('fastTouch browserify error'))
    .pipe(gulp.dest(heimDest))
})

gulp.task('embed-js', () => {
  return embedBundler({debug: true})
    .bundle()
    .pipe(source('embed.js'))
    .pipe(buffer())
    .pipe(sourcemaps.init({loadMaps: true}))
      .pipe(process.env.NODE_ENV === 'production' ? uglify() : gutil.noop())
    .pipe(sourcemaps.write('./', {includeContent: true}))
    .on('error', handleError('embed browserify error'))
    .pipe(gulp.dest(embedDest))
    .pipe(gzip())
    .pipe(gulp.dest(embedDest))
})

gulp.task('raven-js', ['heim-git-commit', 'heim-js'], () => {
  shell('md5sum build/main.js | cut -d " " -f 1', releaseHash => {
    return browserify('./lib/raven.js')
      .transform(envify(_.extend({
        SENTRY_ENDPOINT: process.env.SENTRY_ENDPOINT,
        HEIM_RELEASE: releaseHash,
      }, heimOptions)))
      .bundle()
      .pipe(source('raven.js'))
      .pipe(buffer())
      .pipe(process.env.NODE_ENV === 'production' ? uglify() : gutil.noop())
      .on('error', handleError('raven browserify error'))
      .pipe(gulp.dest(heimDest))
      .pipe(gzip())
      .pipe(gulp.dest(heimDest))
  })
})

gulp.task('heim-less', () => {
  return gulp.src(['./lib/main.less', './lib/crashed.less', './lib/od.less', './site/*.less'])
    .pipe(less({compress: true}))
    .on('error', handleError('LESS error'))
    .pipe(autoprefixer({cascade: false}))
    .on('error', handleError('autoprefixer error'))
    .pipe(gulp.dest(heimDest))
    .pipe(gzip())
    .pipe(gulp.dest(heimDest))
})

gulp.task('emoji-static', () => {
  const emoji = require('./lib/emoji').default
  const twemojiPath = path.dirname(require.resolve('twemoji')) + '/svg/'
  const leadingZeroes = /^0*/
  const lessSource = _.map(emoji.codes, code => {
    if (!code) {
      return ''
    }
    const twemojiName = code.replace(leadingZeroes, '')
    let emojiPath = './res/emoji/' + twemojiName + '.svg'
    if (!fs.existsSync(emojiPath)) {
      emojiPath = twemojiPath + twemojiName + '.svg'
    }
    return '.emoji-' + code + ' { background-image: data-uri("' + emojiPath + '") }'
  }).join('\n')

  const lessFile = gfile('emoji.less', lessSource, {src: true})
    .pipe(less({compress: true}))
    .pipe(gulp.dest(heimDest))
    .pipe(gzip())
    .pipe(gulp.dest(heimDest))

  const indexFile = gfile('emoji.json', JSON.stringify(emoji.index), {src: true})
    .pipe(gulp.dest(heimDest))

  return merge([lessFile, indexFile])
})

gulp.task('heim-static', () => {
  return gulp.src('./static/**/*')
    .pipe(gulp.dest(heimDest))
})

gulp.task('embed-static', () => {
  return gulp.src('./static/robots.txt')
    .pipe(gulp.dest(embedDest))
})

gulp.task('heim-html', ['heim-git-commit'], () => {
  return gulp.src(['./lib/room.html'])
    .pipe(gtemplate(heimOptions))
    .pipe(gulp.dest(heimDest))
})

gulp.task('embed-html', () => {
  return gulp.src(['./lib/embed.html'])
    .pipe(gulp.dest(embedDest))
})

gulp.task('site-templates', ['heim-git-commit'], () => {
  const page = reload('./site/page.js')
  const pages = [
    'home',
    'about',
    'about/values',
    'about/conduct',
    'about/hosts',
    'about/terms',
    'about/privacy',
    'about/dmca',
  ]

  return merge(_.map(pages, name => {
    const html = page.render(reload('./site/' + name))
    return gfile(name + '.html', html, {src: true})
  }))
    .pipe(gulp.dest(heimDest))
})

gulp.task('email-templates', () => {
  require('./emails/email/injectReactAttributes').default()
  const renderEmail = require('./emails/email/renderEmail').default
  const emails = ['welcome', 'room-invitation', 'room-invitation-welcome', 'password-changed', 'password-reset']

  const htmls = merge(_.map(emails, name => {
    const html = renderEmail(reload('./emails/' + name))
    return gfile(name + '.html', html, {src: true})
  }))

  const txtCommon = reload('./emails/common-txt.js').default
  const txts = merge(_.map(emails, name => {
    return gulp.src('./emails/' + name + '.txt')
      .pipe(gtemplate(txtCommon))
  }))

  return merge(htmls, txts)
    .pipe(gulp.dest(emailDest))
})

gulp.task('email-hdrs', () => {
  return gulp.src('./emails/*.hdr').pipe(gulp.dest(emailDest))
})

gulp.task('email-static', () => {
  return gulp.src('./emails/static/*.png').pipe(gulp.dest(emailDest + '/static'))
})

function watchifyTask(name, bundler, outFile, dest) {
  gulp.task(name, ['build-statics'], () => {
    // via https://github.com/gulpjs/gulp/blob/master/docs/recipes/fast-browserify-builds-with-watchify.md
    const watchBundler = watchify(bundler(watchify.args))

    function rebundle() {
      return watchBundler.bundle()
        .on('error', handleError('JS (' + name + ') error'))
        .pipe(source(outFile))
        .pipe(gulp.dest(dest))
        .pipe(gzip())
        .pipe(gulp.dest(dest))
    }

    watchBundler.on('log', gutil.log.bind(gutil, gutil.colors.green('JS (' + name + ')')))
    watchBundler.on('update', rebundle)
    return rebundle()
  })
}

watchifyTask('heim-watchify', heimBundler, 'main.js', heimDest)
watchifyTask('embed-watchify', embedBundler, 'embed.js', embedDest)

gulp.task('build-emails', ['email-templates', 'email-hdrs', 'email-static'])
gulp.task('build-statics', ['raven-js', 'fast-touch-js', 'heim-less', 'emoji-static', 'heim-static', 'embed-static', 'heim-html', 'embed-html', 'site-templates'])
gulp.task('build-browserify', ['heim-js', 'embed-js'])

gulp.task('watch', () => {
  watching = true
  gulp.watch('./lib/**/*.less', ['heim-less'])
  gulp.watch('./res/**/*', ['heim-less', 'emoji-static'])
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
  middleware: function serveHeim(req, res, next) {
    req.url = req.url.replace(/^\/static\/?|^\/room\/\w+\/?/, '/')
    if (req.url === '/') {
      req.url = '/room.html'
    }
    next()
  },
}))

gulp.task('serve-embed', serve({
  port: 8081,
  root: embedDest,
  middleware: function serveEmbed(req, res, next) {
    req.url = req.url.replace(/^\/\?\S+/, '/embed.html')
    next()
  },
}))

gulp.task('develop', ['serve-heim', 'serve-embed', 'default'])
