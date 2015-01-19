var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')

describe('socket store', function() {
  var socket = require('../lib/stores/socket')
  var realWebSocket = window.WebSocket
  var fakeWebSocket, fakeWebSocketContructor

  beforeEach(function() {
    fakeWebSocketContructor = sinon.spy(function() {
      fakeWebSocket = this
      this.send = sinon.spy()
    })
    window.WebSocket = fakeWebSocketContructor
    support.resetStore(socket.store)
  })

  afterEach(function() {
    window.WebSocket = realWebSocket
  })

  describe('_wsurl', function() {
    it('should return wss:host/room/name/ws if protocol is https', function() {
      var loc = {protocol: 'https:', host: 'host', pathname: '/path/'}
      assert.equal(socket.store._wsurl(loc, 'ezzie'), 'wss:host/room/ezzie/ws')
    })
    it('should return ws:host/room/name/ws if protocol is NOT https', function() {
      var loc = {protocol: 'http:', host: 'host', pathname: '/path/'}
      assert.equal(socket.store._wsurl(loc, 'ezzie'), 'ws:host/room/ezzie/ws')
    })
  })

  describe('connect action', function() {
    it('should connect to ws:host/room/name/ws with heim1 protocol', function() {
      socket.store.connect('ezzie')
      var expectedPath = 'ws:' + location.host + '/room/ezzie/ws'
      sinon.assert.calledWithExactly(fakeWebSocketContructor, expectedPath, 'heim1')
    })

    it('should set up event handlers', function() {
      socket.store.connect()
      assert.equal(fakeWebSocket.onopen, socket.store._open)
      assert.equal(fakeWebSocket.onclose, socket.store._close)
      assert.equal(fakeWebSocket.onmessage, socket.store._message)
    })
  })

  describe('when socket opened', function() {
    it('should emit an open event', function(done) {
      support.listenOnce(socket.store, function(ev) {
        assert.deepEqual(ev, {status: 'open'})
        done()
      })

      socket.store._open()
    })
  })

  describe('when socket closed', function() {
    it('should emit an close event', function(done) {
      support.listenOnce(socket.store, function(ev) {
        assert.deepEqual(ev, {status: 'close'})
        done()
      })

      socket.store._close()
    })

    describe('while connected', function() {
      beforeEach(function() {
        socket.store.connect('ezzie')
        sinon.stub(socket.store, 'connect')
      })

      afterEach(function() {
        socket.store.connect.restore()
      })

      it('should attempt to reconnect within 5s', function() {
        socket.store._close()
        support.clock.tick(5000)
        sinon.assert.calledOnce(socket.store.connect)
        sinon.assert.calledWithExactly(socket.store.connect, 'ezzie')
      })
    })
  })

  describe('when message received', function() {
    it('should emit a receive event', function(done) {
      var testBody = {it: 'works'}

      support.listenOnce(socket.store, function(ev) {
        assert.deepEqual(ev, {
          status: 'receive',
          body: testBody
        })
        done()
      })

      socket.store._message({data: JSON.stringify(testBody)})
    })
  })

  describe('send action', function() {
    beforeEach(function() {
      socket.store.connect()
    })

    it('should send JSON to the websocket', function() {
      socket.store.send({
        type: 'send',
        data: {
          content: 'hello, ezzie.',
        }
      })
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'send',
        data: {
          content: 'hello, ezzie.',
        },
        id: '0',
      }))
    })

    it('should increment sequence number', function() {
      function testData(num) {
        return {data: {seqShouldBe: num}}
      }

      function testSent(num) {
        return JSON.stringify({data: {seqShouldBe: num}, id: String(num)})
      }

      socket.store.send(testData(0))
      socket.store.send(testData(1))
      socket.store.send(testData(2))

      sinon.assert.calledWith(fakeWebSocket.send, testSent(0))
      sinon.assert.calledWith(fakeWebSocket.send, testSent(1))
      sinon.assert.calledWith(fakeWebSocket.send, testSent(2))
    })

    it('should send a data property even if unset', function() {
      socket.store.send({})
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({id: '0', data: {}}))
    })
  })
})
