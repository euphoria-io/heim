import support from './support/setup'
import assert from 'assert'
import sinon from 'sinon'

import storage from '../lib/stores/storage'


describe('storage store', () => {
  let clock
  const getItem = localStorage.getItem
  const setItem = localStorage.setItem
  let fakeStorage

  beforeEach(() => {
    clock = support.setupClock()
    fakeStorage = {}
    sinon.stub(localStorage, 'getItem', key => {
      return fakeStorage[key]
    })
    sinon.stub(localStorage, 'setItem', (key, value) => {
      fakeStorage[key] = value
    })
    support.resetStore(storage.store)
  })

  afterEach(() => {
    clock.restore()

    // stub.restore() seems to fail here.
    localStorage.getItem = getItem
    localStorage.setItem = setItem
  })

  describe('load action', () => {
    it('should be synchronous', () => {
      assert.equal(storage.load.sync, true)
    })

    it('should load JSON from localStorage upon load with default empty room index', done => {
      fakeStorage.data = JSON.stringify({it: 'works'})

      support.listenOnce(storage.store, state => {
        assert.deepEqual(state, {it: 'works', room: {}})
        done()
      })

      storage.store.load()
    })

    it('should only load once', () => {
      storage.store.load()
      storage.store.load()
      sinon.assert.calledOnce(localStorage.getItem)
    })
  })

  describe('set action', () => {
    const testKey = 'testKey'
    const testValue = {test: true}

    beforeEach(() => {
      fakeStorage.data = JSON.stringify({testKey: {foo: 'bar'}})
      storage.store.load()
    })

    it('should save JSON to localStorage', () => {
      storage.store.set(testKey, testValue)
      clock.tick(1000)
      sinon.assert.calledWithExactly(localStorage.setItem, 'data', JSON.stringify({
        'testKey': testValue,
        'room': {},
      }))
    })

    it('should not save unchanged values', () => {
      storage.store.set(testKey, testValue)
      clock.tick(1000)
      localStorage.setItem.reset()
      storage.store.set(testKey, testValue)
      clock.tick(1000)
      sinon.assert.notCalled(localStorage.setItem)
    })

    it('should trigger an update event', done => {
      support.listenOnce(storage.store, state => {
        assert.equal(state[testKey], testValue)
        done()
      })

      storage.store.set(testKey, testValue)
    })
  })

  describe('setRoom action', () => {
    const testRoom = 'ezzie'
    const testKey = 'testKey'
    const testValue = {test: true}

    it('should save JSON to localStorage', () => {
      fakeStorage.data = JSON.stringify({room: {ezzie: {testKey: {foo: 'bar'}}}})
      storage.store.load()

      storage.store.setRoom(testRoom, testKey, testValue)
      clock.tick(1000)
      sinon.assert.calledWithExactly(localStorage.setItem, 'data', JSON.stringify({
        'room': {
          'ezzie': {
            'testKey': testValue,
          },
        },
      }))
    })

    it('should not save unchanged values', () => {
      fakeStorage.data = JSON.stringify({})
      storage.store.load()

      storage.store.setRoom(testRoom, testKey, testValue)
      clock.tick(1000)
      localStorage.setItem.reset()
      storage.store.setRoom(testRoom, testKey, testValue)
      clock.tick(1000)
      sinon.assert.notCalled(localStorage.setItem)
    })

    it('should create room config object and trigger an update event', done => {
      fakeStorage.data = JSON.stringify({})
      storage.store.load()

      support.listenOnce(storage.store, state => {
        assert.deepEqual(state.room.ezzie[testKey], testValue)
        done()
      })

      storage.store.setRoom(testRoom, testKey, testValue)
    })
  })

  describe('receiving a storage event', () => {
    it('should be ignored before storage loaded', () => {
      sinon.stub(storage.store, 'trigger')
      storage.store.storageChange({key: 'data', newValue: 'early'})
      assert.equal(storage.store.state, null)
      sinon.assert.notCalled(storage.store.trigger)
      storage.store.trigger.restore()
    })
  })

  describe('receiving a storage event', () => {
    beforeEach(() => {
      fakeStorage.data = JSON.stringify({})
      storage.store.load()
    })

    it('should update state and trigger an update', done => {
      support.listenOnce(storage.store, state => {
        assert.equal(state.hello, 'ezzie')
        done()
      })
      storage.store.storageChange({key: 'data', newValue: JSON.stringify({'hello': 'ezzie'})})
    })

    it('should ignore changes to unknown storage keys', () => {
      sinon.stub(storage.store, 'trigger')
      storage.store.storageChange({key: 'ezzie', newValue: 'bark!'})
      sinon.assert.notCalled(storage.store.trigger)
      storage.store.trigger.restore()
    })

    it('should not trigger an update if unchanged', () => {
      storage.store.set('hello', 'ezzie')
      sinon.stub(storage.store, 'trigger')
      storage.store.storageChange({key: 'data', newValue: JSON.stringify({'hello': 'ezzie'})})
      sinon.assert.notCalled(storage.store.trigger)
      storage.store.trigger.restore()
    })

    it('should not change dirty values pending save', done => {
      storage.store.set('hello', {to: 'ezzie'})
      support.listenOnce(storage.store, state => {
        assert.deepEqual(state.hello, {to: 'ezzie', from: 'max'})
        done()
      })
      storage.store.storageChange({key: 'data', newValue: JSON.stringify({'hello': {from: 'max'}, 'test': 'abcdef'})})
    })

    it('should change previously dirty values after a save', done => {
      storage.store.set('hello', 'ezzie')
      clock.tick(1000)
      support.listenOnce(storage.store, state => {
        assert.equal(state.hello, 'max')
        done()
      })
      storage.store.storageChange({key: 'data', newValue: JSON.stringify({'hello': 'max'})})
    })
  })

  describe('when storage unavailable or disabled', () => {
    beforeEach(() => {
      localStorage.getItem = sinon.stub.throws()
      localStorage.setItem = sinon.stub.throws()
      sinon.stub(console, 'warn')  // eslint-disable-line no-console
    })

    afterEach(() => {
      console.warn.restore()  // eslint-disable-line no-console
    })

    describe('load action', () => {
      it('should initialize with empty store data and room index', done => {
        support.listenOnce(storage.store, state => {
          assert.deepEqual(state, {room: {}})
          done()
        })

        storage.store.load()
      })

      it('should log a warning', () => {
        storage.store.load()
        sinon.assert.calledOnce(console.warn)  // eslint-disable-line no-console
      })
    })

    describe('set action', () => {
      beforeEach(() => {
        storage.store.load()
        console.warn.reset()  // eslint-disable-line no-console
      })

      it('should log a warning', () => {
        storage.store.set('key', 'value')
        clock.tick(1000)
        sinon.assert.calledOnce(console.warn)  // eslint-disable-line no-console
      })
    })
  })
})
