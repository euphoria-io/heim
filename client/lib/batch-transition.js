var _ = require('lodash')


/* This object animates transitions, aggressively batching and downstepping
 * framerate to reduce redraws. When animating slowly changing CSS properties,
 * this can significantly reduce CPU/GPU usage. */

module.exports = function BatchTransition() {
  this._transitions = []
  this._timeout = null
}

_.extend(module.exports.prototype, {
  add: function(transition) {
    var now = Date.now()
    transition.start = now + (transition.startOffset || 0)
    transition._nextFrame = null
    transition._lastValue = null
    this._transitions.push(transition)
    this._run(now)
  },

  _run: function(now) {
    now = now || Date.now()
    var nextFrame = Number.MAX_VALUE
    var maxFPS = 0

    var toReap = 0
    _.each(this._transitions, transition => {
      // skip finished but not reaped transitions
      if (transition.finished) {
        toReap++
        return
      }

      // check for finished transitions
      var t = now - transition.start
      if (t > transition.duration) {
        transition.step(1)
        transition.finished = true
        return
      }

      maxFPS = Math.max(maxFPS, transition.fps)

      // skip transitions that don't need a frame yet
      // fudge the times (and attempt to line up future frames) by slack (default 4) frames
      if (transition._nextFrame && transition._nextFrame > now + (1000 / transition.fps) * (transition.slack || 4)) {
        nextFrame = Math.min(nextFrame, transition._nextFrame)
        return
      }

      // ok, this transition needs a frame. run it
      var value = transition.ease(t / transition.duration)
      transition._lastValue = value
      transition.step(value)

      // figure out when the next frame should be
      // align starting delta frames so animations of the same framerate sync up better
      var delta = Math.floor(now / transition.fps) * transition.fps - now
      while (delta < 1000) {
        delta += 1000 / transition.fps
        value = transition.ease((t + delta) / transition.duration)
        if (transition.shouldStep(value, transition._lastValue)) {
          break
        }
      }
      transition._nextFrame = now + delta

      nextFrame = Math.min(nextFrame, transition._nextFrame)
    })

    // if we have a lot of finished transitions hanging around, clean them up
    if (toReap >= 10) {
      this._transitions = _.filter(this._transitions, transition => !transition.finished)
    }

    if (nextFrame != Number.MAX_VALUE) {
      uiwindow.clearTimeout(this._timeout)
      var delay = Math.max(maxFPS, nextFrame - now)
      this._timeout = uiwindow.setTimeout(() => this._run(), delay)
    } else {
      this._timeout = null
    }
  },
})
