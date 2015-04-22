var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')


describe('activity store', function() {
  var activity = require('../lib/stores/activity')
  var storage = require('../lib/stores/storage')
  var clock

  var roomName = 'space'

  beforeEach(function() {
    clock = support.setupClock()
    sinon.stub(storage, 'setRoom')
    sinon.stub(activity, 'becameActive')
    sinon.stub(activity, 'becameInactive')
    support.resetStore(activity.store)
  })

  afterEach(function() {
    clock.restore()
    activity.becameActive.restore()
    activity.becameInactive.restore()
    storage.setRoom.restore()
  })

  it('should initialize with window unfocused and inactive', function() {
    var initialState = activity.store.getInitialState()
    assert.equal(initialState.windowFocused, false)
    assert.equal(initialState.focusChangedAt, null)
    assert.equal(initialState.active, false)
  })

  describe('when window focused', function() {
    it('should set window state focused', function(done) {
      support.listenOnce(activity.store, function(state) {
        assert.equal(state.windowFocused, true)
        assert.equal(state.focusChangedAt, Date.now())
        done()
      })
      activity.store.windowFocused()
    })
  })

  describe('when window blurred', function() {
    it('should set window state not focused', function(done) {
      support.listenOnce(activity.store, function(state) {
        assert.equal(state.windowFocused, false)
        assert.equal(state.focusChangedAt, Date.now())
        done()
      })
      activity.store.windowBlurred()
    })

    describe('if inactive before', function() {
      it('should not trigger becameInactive', function() {
        assert.equal(activity.store.state.active, false)
        activity.store.windowBlurred()
        sinon.assert.notCalled(activity.becameInactive)
      })
    })
  })

  describe('when page interacted with', function() {
    it('should set window state active and trigger becameActive', function() {
      support.listenOnce(activity.store, function(state) {
        assert.equal(state.active, true)
      })
      activity.store.touch(roomName)
      clock.tick(0)
      sinon.assert.calledOnce(activity.becameActive)
    })

    describe('if active before', function() {
      it('should not trigger becameActive', function() {
        activity.store.touch()
        clock.tick(1000)
        activity.becameActive.reset()
        activity.store.touch(roomName)
        clock.tick(0)
        sinon.assert.notCalled(activity.becameActive)
      })
    })

    it('should after ' + activity.store.idleTime + 'ms set inactive and trigger becameInactive', function() {
      activity.store.touch(roomName)
      support.listenOnce(activity.store, function(state) {
        assert.equal(state.active, false)
      })
      clock.tick(activity.store.idleTime)
      sinon.assert.calledOnce(activity.becameInactive)
    })
  })

  describe('activity history storage', function() {
    it('should be read when storage changes', function(done) {
      support.listenOnce(activity.store, function(state) {
        assert.equal(state.lastActive[roomName], 4321)
        assert.equal(state.lastVisit[roomName], 1234)
        done()
      })
      // TODO: es6
      var mockStorage = {room: {}}
      mockStorage.room[roomName] = {
        lastActive: 4321,
        lastVisit: 1234,
      }
      activity.store.storageChange(mockStorage)
    })

    it('should be written within ' + activity.store.flushTime + 'ms of activity', function() {
      activity.store.touch(roomName)
      clock.tick(1000)
      var lastTouchTime = Date.now()
      activity.store.touch(roomName)
      clock.tick(activity.store.flushTime)
      sinon.assert.calledOnce(storage.setRoom)
      sinon.assert.calledWithExactly(storage.setRoom, roomName, 'lastActive', lastTouchTime)
    })

    it('should update last visit if ' + activity.store.absenceTime + 'ms of absence since last interaction', function() {
      var firstActive = Date.now()
      // TODO: es6
      var mockStorage = {room: {}}
      mockStorage.room[roomName] = {
        lastActive: firstActive,
        lastVisit: null,
      }
      activity.store.storageChange(mockStorage)
      clock.tick(activity.store.absenceTime)
      var lastTouchTime = Date.now()
      storage.setRoom.reset()
      activity.store.touch(roomName)
      clock.tick(activity.store.flushTime)
      sinon.assert.calledTwice(storage.setRoom)
      sinon.assert.calledWithExactly(storage.setRoom, roomName, 'lastActive', lastTouchTime)
      sinon.assert.calledWithExactly(storage.setRoom, roomName, 'lastVisit', firstActive)
    })
  })
})
