var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')

describe('socket store', function() {
  var socket = require('../lib/stores/socket')
  var clock
  var realWebSocket = window.WebSocket
  var fakeWebSocket, fakeWebSocketContructor

  beforeEach(function() {
    clock = support.setupClock()
    fakeWebSocketContructor = sinon.spy(function() {
      fakeWebSocket = this
      this.send = sinon.spy()
      this.close = sinon.spy()
    })
    window.WebSocket = fakeWebSocketContructor
    support.resetStore(socket.store)
  })

  afterEach(function() {
    clock.restore()
    window.WebSocket = realWebSocket
  })

  describe('_wsurl', function() {
    it('should return wss://host/room/name/ws if protocol is https', function() {
      var loc = {protocol: 'https:', host: 'host', pathname: '/path/'}
      assert.equal(socket.store._wsurl(loc, 'ezzie'), 'wss://host/room/ezzie/ws')
    })

    it('should return ws://host/room/name/ws if protocol is NOT https', function() {
      var loc = {protocol: 'http:', host: 'host', pathname: '/path/'}
      assert.equal(socket.store._wsurl(loc, 'ezzie'), 'ws://host/room/ezzie/ws')
    })
  })

  describe('connect action', function() {
    it('should connect to ws://host/room/name/ws with heim1 protocol', function() {
      socket.store.connect('ezzie')
      var expectedPath = 'ws://' + location.host + '/room/ezzie/ws'
      sinon.assert.calledWithExactly(fakeWebSocketContructor, expectedPath, 'heim1')
    })

    it('should set up event handlers', function() {
      socket.store.connect('ezzie')
      assert.equal(fakeWebSocket.onopen, socket.store._open)
      assert.equal(fakeWebSocket.onclose, socket.store._closeReconnectSlow)
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

  function checkSocketCleanup(action) {
    it('should emit an close event', function(done) {
      support.listenOnce(socket.store, function(ev) {
        assert.deepEqual(ev, {status: 'close'})
        done()
      })

      action()
    })

    it('should clean up timeouts', function() {
      var pingTimeout = socket.store.pingTimeout = 1
      var pingReplyTimeout = socket.store.pingReplyTimeout = 2
      sinon.stub(window, 'clearTimeout')
      action()
      sinon.assert.calledTwice(window.clearTimeout)
      sinon.assert.calledWithExactly(window.clearTimeout, pingTimeout)
      sinon.assert.calledWithExactly(window.clearTimeout, pingReplyTimeout)
      window.clearTimeout.restore()
    })

    it('should clear socket event handlers', function() {
      action()
      assert.equal(fakeWebSocket.onopen, null)
      assert.equal(fakeWebSocket.onclose, null)
      assert.equal(fakeWebSocket.onmessage, null)
    })
  }

  describe('when socket closed', function() {
    beforeEach(function() {
      socket.store.connect('ezzie')
      sinon.stub(socket.store, '_connect')
    })

    afterEach(function() {
      socket.store._connect.restore()
    })

    checkSocketCleanup(() => socket.store.ws.onclose())

    it('should attempt to reconnect within 5s', function() {
      socket.store.ws.onclose()
      clock.tick(5000)
      sinon.assert.calledOnce(socket.store._connect)
    })
  })

  describe('a forceful reconnect', function() {
    beforeEach(function() {
      socket.store.connect('ezzie')
      sinon.stub(socket.store, '_connect')
    })

    afterEach(function() {
      socket.store._connect.restore()
    })

    checkSocketCleanup(socket.store._reconnect)

    it('should close the socket and connect again', function() {
      socket.store._reconnect()
      sinon.assert.calledOnce(socket.store.ws.close)
      sinon.assert.calledOnce(socket.store._connect)
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

  describe('when server ping received', function() {
    beforeEach(function() {
      sinon.spy(window, 'setTimeout')
      socket.store.connect('ezzie')
      socket.store._message({data: JSON.stringify({
        type: 'ping-event',
        data: {
          time: 0,
          next: 20,
        },
      })})
    })

    afterEach(function() {
      window.setTimeout.restore()
    })

    it('should send a ping-reply', function() {
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping-reply',
        data: {
          time: 0
        },
        id: '0',
      }))
    })

    it('should schedule timeout', function() {
      sinon.assert.calledWith(setTimeout, socket.store._ping, 20 * 1000)
    })

    describe('when a second ping received late', function() {
      beforeEach(function() {
        setTimeout.reset()
        socket.store._message({data: JSON.stringify({
          type: 'ping-event',
          data: {
            time: 0,
            next: 10,
          },
        })})
      })

      it('should not schedule timeout', function() {
        sinon.assert.notCalled(setTimeout)
      })
    })

    describe('if another server ping isn\'t received before the next timeout', function() {
      beforeEach(function() {
        fakeWebSocket.send.reset()
        sinon.stub(socket.store, '_reconnect')
        clock.tick(20000)
      })

      afterEach(function() {
        socket.store._reconnect.restore()
        clearTimeout(socket.store.pingTimeout)
        clearTimeout(socket.store.pingReplyTimeout)
      })

      it('should send a client ping', function() {
        sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
          type: 'ping',
          id: '1',
          data: {},
        }))
      })

      describe('after 2000ms', function() {
        describe('if there is no response', function() {
          it('should force a reconnect', function() {
            clock.tick(2000)
            sinon.assert.calledOnce(socket.store._reconnect)
          })
        })

        describe('if any server message received', function() {
          it('should not reconnect', function() {
            clock.tick(1000)
            socket.store._message({data: JSON.stringify({
              type: 'another-message',
            })})
            clock.tick(1000)
            sinon.assert.notCalled(socket.store._reconnect)
          })
        })
      })
    })
  })

  describe('pingIfIdle action', function() {
    beforeEach(function() {
      socket.store.connect('ezzie')
    })

    it('should send a ping if no messages have ever been received', function() {
      socket.store.pingIfIdle()
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping',
        id: '0',
        data: {},
      }))
    })

    it('should send a ping if no messages have been received in the last 2000ms', function() {
      socket.store._message({data: JSON.stringify({
        type: 'hello, ezzie.',
      })})
      clock.tick(2000)
      socket.store.pingIfIdle()
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping',
        id: '0',
        data: {},
      }))
    })

    it('should not send a ping if a message has been received in the last 2000ms', function() {
      socket.store._message({data: JSON.stringify({
        type: 'hello, ezzie.',
      })})
      clock.tick(1000)
      socket.store.pingIfIdle()
      sinon.assert.notCalled(fakeWebSocket.send)
    })

    it('should not send a second ping if one was sent in the last 2000ms', function() {
      socket.store.pingIfIdle()
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping',
        id: '0',
        data: {},
      }))
      fakeWebSocket.send.reset()
      clock.tick(100)
      socket.store.pingIfIdle()
      sinon.assert.notCalled(fakeWebSocket.send)
    })
  })

  describe('send action', function() {
    beforeEach(function() {
      socket.store.connect('ezzie')
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

    it('should send an id property if specified', function() {
      socket.store.send({id: 'heyyy'})
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({id: 'heyyy', data: {}}))
    })
  })
})
