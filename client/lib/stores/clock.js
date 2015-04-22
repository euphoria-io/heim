var Reflux = require('reflux')


var second = module.exports.second = Reflux.createStore({
  init: function() {
    this.state = Date.now()
    this._tick()
  },

  getInitialState: function() {
    return this.state
  },

  _tick: function() {
    var now = Date.now()
    this.state = now
    this.trigger(this.state)
    setTimeout(this._tick, 1000 - (now % 1000))
  },
})


module.exports.minute = Reflux.createStore({
  listenables: [
    {secondTick: second},
  ],

  init: function() {
    this.state = Date.now()
    this._last = this.state
  },

  getInitialState: function() {
    return this.state
  },

  secondTick: function(now) {
    // detect execution pauses / suspends
    var skipped = now > this._last + 2000
    this._last = now

    var onMinute = Math.floor(now / 1000) % 60 === 0
    if (skipped || onMinute) {
      this.state = now
      this.trigger(this.state)
    }
  },
})
