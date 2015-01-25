var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')


describe('storage store', function() {
  var storage = require('../lib/stores/storage')
  var getItem = localStorage.getItem
  var setItem = localStorage.setItem
  var fakeStorage

  beforeEach(function() {
    fakeStorage = {}
    sinon.stub(localStorage, 'getItem', function(key) {
      return fakeStorage[key]
    })
    sinon.stub(localStorage, 'setItem', function(key, value) {
      fakeStorage[key] = value
    })
    support.resetStore(storage.store)
  })

  afterEach(function() {
    // stub.restore() seems to fail here.
    localStorage.getItem = getItem
    localStorage.setItem = setItem
  })

  describe('load action', function() {
    it('should be synchronous', function() {
      assert.equal(storage.load.sync, true)
    })

    it('should load JSON from localStorage upon load with default empty room index', function(done) {
      fakeStorage.data = JSON.stringify({it: 'works'})

      support.listenOnce(storage.store, function(state) {
        assert.deepEqual(state, {it: 'works', room: {}})
        done()
      })

      storage.store.load()
    })

    it('should only load once', function() {
      storage.store.load()
      storage.store.load()
      sinon.assert.calledOnce(localStorage.getItem)
    })
  })

  describe('set action', function() {
    var testKey = 'testKey'
    var testValue = {test: true}

    beforeEach(function() {
      fakeStorage.data = JSON.stringify({})
      storage.store.load()
    })

    it('should save JSON to localStorage', function() {
      storage.store.set(testKey, testValue)
      support.clock.tick(1000)
      sinon.assert.calledWithExactly(localStorage.setItem, 'data', JSON.stringify({
        'room': {},
        'testKey': testValue,
      }))
    })

    it('should trigger an update event', function(done) {
      support.listenOnce(storage.store, function(state) {
        assert.equal(state[testKey], testValue)
        done()
      })

      storage.store.set(testKey, testValue)
    })
  })

  describe('setRoom action', function() {
    var testRoom = 'ezzie'
    var testKey = 'testKey'
    var testValue = {test: true}

    beforeEach(function() {
      fakeStorage.data = JSON.stringify({})
      storage.store.load()
    })

    it('should save JSON to localStorage', function() {
      storage.store.setRoom(testRoom, testKey, testValue)
      support.clock.tick(1000)
      sinon.assert.calledWithExactly(localStorage.setItem, 'data', JSON.stringify({
        'room': {
          'ezzie': {
            'testKey': testValue
          }
        }
      }))
    })

    it('should create room config object and trigger an update event', function(done) {
      support.listenOnce(storage.store, function(state) {
        assert.deepEqual(state.room.ezzie[testKey], testValue)
        done()
      })

      storage.store.setRoom(testRoom, testKey, testValue)
    })
  })

  describe('receiving a storage event', function() {
    beforeEach(function() {
      fakeStorage.data = JSON.stringify({})
      storage.store.load()
    })

    it('should update state and trigger an update', function(done) {
      support.listenOnce(storage.store, function(state) {
        assert.equal(state.hello, 'ezzie')
        done()
      })
      storage.store.onStorageUpdate({key: 'data', newValue: JSON.stringify({'hello': 'ezzie'})})
    })

    it('should ignore changes to unknown storage keys', function() {
      sinon.stub(storage.store, 'trigger')
      storage.store.onStorageUpdate({key: 'ezzie', newValue: 'bark!'})
      sinon.assert.notCalled(storage.store.trigger)
      storage.store.trigger.restore()
    })

    it('should not trigger an update if unchanged', function() {
      storage.store.set('hello', 'ezzie')
      sinon.stub(storage.store, 'trigger')
      storage.store.onStorageUpdate({key: 'data', newValue: JSON.stringify({'hello': 'ezzie'})})
      sinon.assert.notCalled(storage.store.trigger)
      storage.store.trigger.restore()
    })

    it('should not change dirty values pending save', function(done) {
      storage.store.set('hello', 'ezzie')
      support.listenOnce(storage.store, function(state) {
        assert.equal(state.hello, 'ezzie')
        done()
      })
      storage.store.onStorageUpdate({key: 'data', newValue: JSON.stringify({'hello': 'max', 'test': 'abcdef'})})
    })

    it('should change previously dirty values after a save', function(done) {
      storage.store.set('hello', 'ezzie')
      support.clock.tick(1000)
      support.listenOnce(storage.store, function(state) {
        assert.equal(state.hello, 'max')
        done()
      })
      storage.store.onStorageUpdate({key: 'data', newValue: JSON.stringify({'hello': 'max'})})
    })
  })
})
