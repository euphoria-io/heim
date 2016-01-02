import 'babel-polyfill'
import sinon from 'sinon'
import Immutable from 'immutable'
import Reflux from 'reflux'
import _ from 'lodash'


Immutable.Iterable.noLengthWarning = true
Reflux.nextTick(callback => window.setTimeout(callback, 0))

export function setupClock() {
  const clock = sinon.useFakeTimers()

  // manually fix Sinon #624 until it updates Lolex to 1.2.0
  Date.now = () => { return Date().getTime() }

  // set up fake clock to work with lodash

  const origDebounce = _.debounce
  const origThrottle = _.throttle

  const mock_ = _.runInContext(window)
  _.debounce = mock_.debounce
  _.throttle = mock_.throttle

  const origRestore = clock.restore.bind(clock)
  clock.restore = () => {
    _.debounce = origDebounce
    _.throttle = origThrottle
    origRestore()
  }

  // remove erroneous entry from coverage listing
  Date.now()

  // start with an initial time (_.throttle seems to need the starting
  // clock to be greater than the throttle period)
  clock.tick(60 * 60 * 1000)

  return clock
}

export function listenOnce(listenable, callback) {
  const remove = listenable.listen(function handleOnce() {
    remove()
    callback.apply(this, arguments)
  })
}

export function resetStore(store) {
  store.init()
  store.emitter.removeAllListeners()
}

export function fakeEnv(env) {
  let origProcessEnv

  before(() => {
    origProcessEnv = process.env
    process.env = env
  })

  after(() => {
    process.env = origProcessEnv
  })
}

window.Heim = {
  setFavicon: () => {},
  setTitleMsg: () => {},
  setTitlePrefix: () => {},
}

export default { setupClock, listenOnce, resetStore, fakeEnv }
