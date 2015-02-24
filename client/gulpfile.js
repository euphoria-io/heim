var gulp = require('gulp')
var gutil = require('gulp-util')
var streamify = require('gulp-streamify')
var less = require('gulp-less')
var uglify = require('gulp-uglify')
var source = require('vinyl-source-stream')
var watchify = require('watchify')
var browserify = require('browserify')
var react = require('gulp-react')
var jshint = require('gulp-jshint')


var dest = './build'

function bundler(args) {
  return browserify('./lib/client.js', args)
}

gulp.task('js', function() {
  return bundler()
    // share some libraries with the global namespace
    // doing this here because these exposes trip up watchify atm
    .require('lodash', {expose: 'lodash'})
    .require('react', {expose: 'react'})
    .require('reflux', {expose: 'reflux'})
    .require('immutable', {expose: 'immutable'})
    .require('moment', {expose: 'moment'})
    .bundle()
    .pipe(source('main.js'))
    .pipe(process.env.NODE_ENV == 'production' ? streamify(uglify()) : gutil.noop())
    .on('error', gutil.log.bind(gutil, 'browserify error'))
    .pipe(gulp.dest(dest))
})

gulp.task('less', function() {
  return gulp.src('./lib/main.less')
    .pipe(less({compress: true}))
    .on('error', function(err) {
      gutil.log(gutil.colors.red('LESS error:'), err.message)
      this.emit('end')
    })
    .pipe(gulp.dest(dest))
})

gulp.task('static', function() {
  return gulp.src('./static/**/*')
    .pipe(gulp.dest(dest))
})

gulp.task('html', function() {
  return gulp.src('./lib/index.html')
    .pipe(gulp.dest(dest))
})

gulp.task('lint', function() {
  return gulp.src(['./lib/**/*.js', './test/**/*.js', './gulpfile.js'])
    .pipe(react())
    .pipe(jshint())
    .pipe(jshint.reporter('default'))
    .pipe(jshint.reporter('fail'))
})

gulp.task('watchify', function() {
  // via https://github.com/gulpjs/gulp/blob/master/docs/recipes/fast-browserify-builds-with-watchify.md
  bundler = watchify(bundler(watchify.args))
  bundler.on('log', gutil.log.bind(gutil, gutil.colors.green('Watchify')))
  bundler.on('update', rebundle)

  function rebundle() {
    return bundler.bundle()
      .on('error', function(err) {
        gutil.log(gutil.colors.red('Watchify error:'), err.message)
      })
      .pipe(source('main.js'))
      .pipe(gulp.dest(dest))
  }

  return rebundle()
})

gulp.task('watch', function () {
  gulp.watch('./lib/**/*.less', ['less'])
  gulp.watch('./res/**/*', ['less'])
  gulp.watch('./lib/**/*.html', ['html'])
  gulp.watch('./static/**/*', ['static'])
})

gulp.task('build', ['js', 'less', 'static', 'html'])
gulp.task('default', ['less', 'static', 'html', 'watch', 'watchify'])
