var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')


describe('storage store', function() {
  var storage = require('../lib/stores/storage')
  var clock
  var getItem = localStorage.getItem
  var setItem = localStorage.setItem
  var fakeStorage

  beforeEach(function() {
    clock = support.setupClock()
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
    clock.restore()

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
      fakeStorage.data = JSON.stringify({testKey: {foo: 'bar'}})
      storage.store.load()
    })

    it('should save JSON to localStorage', function() {
      storage.store.set(testKey, testValue)
      clock.tick(1000)
      sinon.assert.calledWithExactly(localStorage.setItem, 'data', JSON.stringify({
        'testKey': testValue,
        'room': {},
      }))
    })

    it('should not save unchanged values', function() {
      storage.store.set(testKey, testValue)
      clock.tick(1000)
      localStorage.setItem.reset()
      storage.store.set(testKey, testValue)
      clock.tick(1000)
      sinon.assert.notCalled(localStorage.setItem)
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

    it('should save JSON to localStorage', function() {
      fakeStorage.data = JSON.stringify({room: {ezzie: {testKey: {foo: 'bar'}}}})
      storage.store.load()

      storage.store.setRoom(testRoom, testKey, testValue)
      clock.tick(1000)
      sinon.assert.calledWithExactly(localStorage.setItem, 'data', JSON.stringify({
        'room': {
          'ezzie': {
            'testKey': testValue
          }
        }
      }))
    })

    it('should not save unchanged values', function() {
      fakeStorage.data = JSON.stringify({})
      storage.store.load()

      storage.store.setRoom(testRoom, testKey, testValue)
      clock.tick(1000)
      localStorage.setItem.reset()
      storage.store.setRoom(testRoom, testKey, testValue)
      clock.tick(1000)
      sinon.assert.notCalled(localStorage.setItem)
    })

    it('should create room config object and trigger an update event', function(done) {
      fakeStorage.data = JSON.stringify({})
      storage.store.load()

      support.listenOnce(storage.store, function(state) {
        assert.deepEqual(state.room.ezzie[testKey], testValue)
        done()
      })

      storage.store.setRoom(testRoom, testKey, testValue)
    })
  })

  describe('receiving a storage event', function() {
    it('should be ignored before storage loaded', function() {
      sinon.stub(storage.store, 'trigger')
      storage.store.storageChange({key: 'data', newValue: 'early'})
      assert.equal(storage.store.state, null)
      sinon.assert.notCalled(storage.store.trigger)
      storage.store.trigger.restore()
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
      storage.store.storageChange({key: 'data', newValue: JSON.stringify({'hello': 'ezzie'})})
    })

    it('should ignore changes to unknown storage keys', function() {
      sinon.stub(storage.store, 'trigger')
      storage.store.storageChange({key: 'ezzie', newValue: 'bark!'})
      sinon.assert.notCalled(storage.store.trigger)
      storage.store.trigger.restore()
    })

    it('should not trigger an update if unchanged', function() {
      storage.store.set('hello', 'ezzie')
      sinon.stub(storage.store, 'trigger')
      storage.store.storageChange({key: 'data', newValue: JSON.stringify({'hello': 'ezzie'})})
      sinon.assert.notCalled(storage.store.trigger)
      storage.store.trigger.restore()
    })

    it('should not change dirty values pending save', function(done) {
      storage.store.set('hello', {to: 'ezzie'})
      support.listenOnce(storage.store, function(state) {
        assert.deepEqual(state.hello, {to: 'ezzie', from: 'max'})
        done()
      })
      storage.store.storageChange({key: 'data', newValue: JSON.stringify({'hello': {from: 'max'}, 'test': 'abcdef'})})
    })

    it('should change previously dirty values after a save', function(done) {
      storage.store.set('hello', 'ezzie')
      clock.tick(1000)
      support.listenOnce(storage.store, function(state) {
        assert.equal(state.hello, 'max')
        done()
      })
      storage.store.storageChange({key: 'data', newValue: JSON.stringify({'hello': 'max'})})
    })
  })

  describe('when storage unavailable or disabled', function() {
    beforeEach(function() {
      localStorage.getItem = sinon.stub.throws()
      localStorage.setItem = sinon.stub.throws()
      sinon.stub(console, 'warn')
    })

    afterEach(function() {
      console.warn.restore()
    })

    describe('load action', function() {
      it('should initialize with empty store data and room index', function(done) {
        support.listenOnce(storage.store, function(state) {
          assert.deepEqual(state, {room: {}})
          done()
        })

        storage.store.load()
      })

      it('should log a warning', function() {
        storage.store.load()
        sinon.assert.calledOnce(console.warn)
      })
    })

    describe('set action', function() {
      beforeEach(function() {
        storage.store.load()
        console.warn.reset()
      })

      it('should log a warning', function() {
        storage.store.set('key', 'value')
        clock.tick(1000)
        sinon.assert.calledOnce(console.warn)
      })
    })
  })
})
