require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')


describe('EventListeners', function() {
  var EventListeners = require('../lib/eventlisteners')
  var eventType = 'test'
  var eventCallback = function() {}
  var evs

  function fakeTarget() {
    return {
      addEventListener: sinon.stub(),
      removeEventListener: sinon.stub(),
    }
  }

  beforeEach(function() {
    evs = new EventListeners()
  })

  it('should initialize with empty listeners array', function() {
    assert.deepEqual(evs._listeners, [])
  })

  describe('addEventListener', function() {
    it('should add an event listener', function() {
      var target = fakeTarget()
      evs.addEventListener(target, eventType, eventCallback, false)
      sinon.assert.calledOnce(target.addEventListener)
      sinon.assert.calledWithExactly(target.addEventListener, eventType, eventCallback, false)
    })
  })

  describe('removeEventListener', function() {
    it('should remove an event listener', function() {
      var target = fakeTarget()
      evs.removeEventListener(target, eventType, eventCallback, false)
      sinon.assert.calledOnce(target.removeEventListener)
      sinon.assert.calledWithExactly(target.removeEventListener, eventType, eventCallback, false)
    })
  })

  describe('removeAllEventListeners', function() {
    it('should remove all current event listeners', function() {
      var target1 = fakeTarget()
      var target2 = fakeTarget()
      var target3 = fakeTarget()
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
