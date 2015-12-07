require('./support/setup')
import assert from 'assert'
import sinon from 'sinon'

import EventListeners from '../lib/EventListeners'


describe('EventListeners', () => {
  const eventType = 'test'
  const eventCallback = () => {}
  let evs

  function fakeTarget() {
    return {
      addEventListener: sinon.stub(),
      removeEventListener: sinon.stub(),
    }
  }

  beforeEach(() => {
    evs = new EventListeners()
  })

  it('should initialize with empty listeners array', () => {
    assert.deepEqual(evs._listeners, [])
  })

  describe('addEventListener', () => {
    it('should add an event listener', () => {
      const target = fakeTarget()
      evs.addEventListener(target, eventType, eventCallback, false)
      sinon.assert.calledOnce(target.addEventListener)
      sinon.assert.calledWithExactly(target.addEventListener, eventType, eventCallback, false)
    })
  })

  describe('removeEventListener', () => {
    it('should remove an event listener', () => {
      const target = fakeTarget()
      evs.removeEventListener(target, eventType, eventCallback, false)
      sinon.assert.calledOnce(target.removeEventListener)
      sinon.assert.calledWithExactly(target.removeEventListener, eventType, eventCallback, false)
    })
  })

  describe('removeAllEventListeners', () => {
    it('should remove all current event listeners', () => {
      const target1 = fakeTarget()
      const target2 = fakeTarget()
      const target3 = fakeTarget()
      evs.addEventListener(target1, eventType, eventCallback, false)
      evs.addEventListener(target2, eventType, eventCallback, false)
      evs.addEventListener(target3, eventType, eventCallback, false)
      assert.equal(evs._listeners.length, 3)
      evs.removeEventListener(target2, eventType, eventCallback, false)
      assert.equal(evs._listeners.length, 2)
      evs.removeAllEventListeners()
      sinon.assert.calledWithExactly(target1.removeEventListener, eventType, eventCallback, false)
      sinon.assert.calledWithExactly(target3.removeEventListener, eventType, eventCallback, false)
      assert.equal(evs._listeners.length, 0)
    })
  })
})
