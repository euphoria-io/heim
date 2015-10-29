import Reflux from 'reflux'


export const second = Reflux.createStore({
  init() {
    this.state = Date.now()
    this._tick()
  },

  getInitialState() {
    return this.state
  },

  _tick() {
    const now = Date.now()
    this.state = now
    this.trigger(this.state)
    setTimeout(this._tick, 1000 - (now % 1000))
  },
})


export const minute = Reflux.createStore({
  listenables: [
    {secondTick: second},
  ],

  init() {
    this.state = Date.now()
    this._last = this.state
  },

  getInitialState() {
    return this.state
  },

  secondTick(now) {
    // detect execution pauses / suspends
    const skipped = now > this._last + 2000
    this._last = now

    const onMinute = Math.floor(now / 1000) % 60 === 0
    if (skipped || onMinute) {
      this.state = now
      this.trigger(this.state)
    }
  },
})

export default { second, minute }
