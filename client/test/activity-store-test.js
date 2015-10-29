import support from './support/setup'
import assert from 'assert'
import sinon from 'sinon'

import activity from '../lib/stores/activity'
import storage from '../lib/stores/storage'

describe('activity store', () => {
  let clock
  const roomName = 'space'

  beforeEach(() => {
    clock = support.setupClock()
    sinon.stub(storage, 'setRoom')
    sinon.stub(activity, 'becameActive')
    sinon.stub(activity, 'becameInactive')
    support.resetStore(activity.store)
  })

  afterEach(() => {
    clock.restore()
    activity.becameActive.restore()
    activity.becameInactive.restore()
    storage.setRoom.restore()
  })

  it('should initialize with window unfocused and inactive', () => {
    const initialState = activity.store.getInitialState()
    assert.equal(initialState.windowFocused, false)
    assert.equal(initialState.focusChangedAt, null)
    assert.equal(initialState.active, false)
  })

  describe('when window focused', () => {
    it('should set window state focused', done => {
      support.listenOnce(activity.store, state => {
        assert.equal(state.windowFocused, true)
        assert.equal(state.focusChangedAt, Date.now())
        done()
      })
      activity.store.windowFocused()
    })
  })

  describe('when window blurred', () => {
    it('should set window state not focused', done => {
      support.listenOnce(activity.store, state => {
        assert.equal(state.windowFocused, false)
        assert.equal(state.focusChangedAt, Date.now())
        done()
      })
      activity.store.windowBlurred()
    })

    describe('if inactive before', () => {
      it('should not trigger becameInactive', () => {
        assert.equal(activity.store.state.active, false)
        activity.store.windowBlurred()
        sinon.assert.notCalled(activity.becameInactive)
      })
    })
  })

  describe('when page interacted with', () => {
    it('should set window state active and trigger becameActive', () => {
      support.listenOnce(activity.store, state => {
        assert.equal(state.active, true)
      })
      activity.store.touch(roomName)
      clock.tick(0)
      sinon.assert.calledOnce(activity.becameActive)
    })

    describe('if active before', () => {
      it('should not trigger becameActive', () => {
        activity.store.touch()
        clock.tick(1000)
        activity.becameActive.reset()
        activity.store.touch(roomName)
        clock.tick(0)
        sinon.assert.notCalled(activity.becameActive)
      })
    })

    it('should after ' + activity.store.idleTime + 'ms set inactive and trigger becameInactive', () => {
      activity.store.touch(roomName)
      support.listenOnce(activity.store, state => {
        assert.equal(state.active, false)
      })
      clock.tick(activity.store.idleTime)
      sinon.assert.calledOnce(activity.becameInactive)
    })
  })

  describe('activity history storage', () => {
    it('should be read when storage changes', done => {
      support.listenOnce(activity.store, state => {
        assert.equal(state.lastActive[roomName], 4321)
        assert.equal(state.lastVisit[roomName], 1234)
        done()
      })
      // TODO: es6
      const mockStorage = {room: {}}
      mockStorage.room[roomName] = {
        lastActive: 4321,
        lastVisit: 1234,
      }
      activity.store.storageChange(mockStorage)
    })

    it('should be written within ' + activity.store.flushTime + 'ms of activity', () => {
      activity.store.touch(roomName)
      clock.tick(1000)
      const lastTouchTime = Date.now()
      activity.store.touch(roomName)
      clock.tick(activity.store.flushTime)
      sinon.assert.calledOnce(storage.setRoom)
      sinon.assert.calledWithExactly(storage.setRoom, roomName, 'lastActive', lastTouchTime)
    })

    it('should update last visit if ' + activity.store.absenceTime + 'ms of absence since last interaction', () => {
      const firstActive = Date.now()
      // TODO: es6
      const mockStorage = {room: {}}
      mockStorage.room[roomName] = {
        lastActive: firstActive,
        lastVisit: null,
      }
      activity.store.storageChange(mockStorage)
      clock.tick(activity.store.absenceTime)
      const lastTouchTime = Date.now()
      storage.setRoom.reset()
      activity.store.touch(roomName)
      clock.tick(activity.store.flushTime)
      sinon.assert.calledTwice(storage.setRoom)
      sinon.assert.calledWithExactly(storage.setRoom, roomName, 'lastActive', lastTouchTime)
      sinon.assert.calledWithExactly(storage.setRoom, roomName, 'lastVisit', firstActive)
    })
  })
})
