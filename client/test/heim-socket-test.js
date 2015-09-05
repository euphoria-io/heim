var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')


describe('socket store', function() {
  var Socket = require('../lib/heim/socket')
  var clock
  var realWebSocket = window.WebSocket
  var fakeWebSocket, fakeWebSocketContructor
  var socket

  beforeEach(function() {
    clock = support.setupClock()
    fakeWebSocketContructor = sinon.spy(function() {
      fakeWebSocket = this
      this.send = sinon.spy()
      this.close = sinon.spy()
    })
    window.WebSocket = fakeWebSocketContructor
    socket = new Socket()
  })

  afterEach(function() {
    clock.restore()
    window.WebSocket = realWebSocket
  })

  describe('_wsurl', function() {
    it('should return wss://host/room/name/ws?h=1 if protocol is https', function() {
      assert.equal(socket._wsurl('https://host/prefix', 'ezzie'), 'wss://host/prefix/room/ezzie/ws?h=1')
    })

    it('should return ws://host/room/name/ws?h=1 if protocol is NOT https', function() {
      assert.equal(socket._wsurl('http://host/prefix', 'ezzie'), 'ws://host/prefix/room/ezzie/ws?h=1')
    })
  })

  describe('connect method', function() {
    it('should by connect to wss://heimhost/test/ws?h=1 with heim1 protocol', function() {
      socket.connect('https://heimhost/test', 'ezzie')
      var expectedPath = 'wss://heimhost/test/room/ezzie/ws?h=1'
      sinon.assert.calledWithExactly(fakeWebSocketContructor, expectedPath, 'heim1')
    })

    it('should set up event handlers', function() {
      socket.connect('https://heimhost/test', 'ezzie')
      assert(fakeWebSocket.onopen)
      assert(fakeWebSocket.onclose)
      assert(fakeWebSocket.onmessage)
    })
  })

  describe('when socket opened', function() {
    it('should emit an open event', function(done) {
      socket.once('open', done)
      socket._onOpen()
    })
  })

  function checkSocketCleanup(action) {
    it('should emit an close event', function(done) {
      socket.once('close', done)
      action()
    })

    it('should clean up timeouts', function() {
      var pingTimeout = socket.pingTimeout = 1
      var pingReplyTimeout = socket.pingReplyTimeout = 2
      sinon.stub(window, 'clearTimeout')
      action()
      sinon.assert.calledTwice(window.clearTimeout)
      sinon.assert.calledWithExactly(window.clearTimeout, pingTimeout)
      sinon.assert.calledWithExactly(window.clearTimeout, pingReplyTimeout)
      window.clearTimeout.restore()
    })
  }

  describe('when socket closed', function() {
    beforeEach(function() {
      socket.connect('https://heimhost/test', 'ezzie')
      sinon.stub(socket, 'reconnect')
    })

    afterEach(function() {
      socket.reconnect.restore()
    })

    checkSocketCleanup(() => socket.ws.onclose())

    it('should clear socket event handlers', function() {
      socket.ws.onclose()
      assert.equal(fakeWebSocket.onopen, null)
      assert.equal(fakeWebSocket.onclose, null)
      assert.equal(fakeWebSocket.onmessage, null)
    })

    it('should attempt to reconnect within 5s', function() {
      socket.ws.onclose()
      clock.tick(5000)
      sinon.assert.calledOnce(socket.reconnect)
    })
  })

  describe('a forceful reconnect', function() {
    beforeEach(function() {
      socket.connect('https://heimhost/test', 'ezzie')
    })

    checkSocketCleanup(() => socket.reconnect())

    it('should close the socket and connect again', function() {
      var oldWs = socket.ws
      fakeWebSocketContructor.reset()
      socket.reconnect()
      sinon.assert.calledOnce(oldWs.close)
      sinon.assert.calledOnce(fakeWebSocketContructor)
    })
  })

  describe('when message received', function() {
    it('should emit a receive event', function(done) {
      var testBody = {it: 'works'}

      socket.once('receive', function(body) {
        assert.deepEqual(body, testBody)
        done()
      })

      socket._onMessage({data: JSON.stringify(testBody)})
    })
  })

  describe('when server ping received', function() {
    beforeEach(function() {
      sinon.spy(window, 'setTimeout')
      socket.connect('https://heimhost/test', 'ezzie')
      socket._onMessage({data: JSON.stringify({
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
      sinon.assert.calledWith(setTimeout, sinon.match.func, 20 * 1000)
    })

    describe('when a second ping received late', function() {
      beforeEach(function() {
        setTimeout.reset()
        socket._onMessage({data: JSON.stringify({
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
        sinon.stub(socket, 'reconnect')
        clock.tick(20000)
      })

      afterEach(function() {
        socket.reconnect.restore()
        clearTimeout(socket.pingTimeout)
        clearTimeout(socket.pingReplyTimeout)
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
            sinon.assert.calledOnce(socket.reconnect)
          })
        })

        describe('if any server message received', function() {
          it('should not reconnect', function() {
            clock.tick(1000)
            socket._onMessage({data: JSON.stringify({
              type: 'another-message',
            })})
            clock.tick(1000)
            sinon.assert.notCalled(socket.reconnect)
          })
        })
      })
    })
  })

  describe('pingIfIdle action', function() {
    beforeEach(function() {
      socket.connect('https://heimhost/test', 'ezzie')
    })

    it('should send a ping if no messages have ever been received', function() {
      socket.pingIfIdle()
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping',
        id: '0',
        data: {},
      }))
    })

    it('should send a ping if no messages have been received in the last 2000ms', function() {
      socket._onMessage({data: JSON.stringify({
        type: 'hello, ezzie.',
      })})
      clock.tick(2000)
      socket.pingIfIdle()
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping',
        id: '0',
        data: {},
      }))
    })

    it('should not send a ping if a message has been received in the last 2000ms', function() {
      socket._onMessage({data: JSON.stringify({
        type: 'hello, ezzie.',
      })})
      clock.tick(1000)
      socket.pingIfIdle()
      sinon.assert.notCalled(fakeWebSocket.send)
    })

    it('should not send a second ping if one was sent in the last 2000ms', function() {
      socket.pingIfIdle()
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping',
        id: '0',
        data: {},
      }))
      fakeWebSocket.send.reset()
      clock.tick(100)
      socket.pingIfIdle()
      sinon.assert.notCalled(fakeWebSocket.send)
    })
  })

  describe('send action', function() {
    beforeEach(function() {
      socket.connect('https://heimhost/test', 'ezzie')
    })

    it('should send JSON to the websocket', function() {
      socket.send({
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

      socket.send(testData(0))
      socket.send(testData(1))
      socket.send(testData(2))

      sinon.assert.calledWith(fakeWebSocket.send, testSent(0))
      sinon.assert.calledWith(fakeWebSocket.send, testSent(1))
      sinon.assert.calledWith(fakeWebSocket.send, testSent(2))
    })

    it('should send a data property even if unset', function() {
      socket.send({})
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({id: '0', data: {}}))
    })

    it('should send an id property if specified', function() {
      socket.send({id: 'heyyy'})
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({id: 'heyyy', data: {}}))
    })
  })

  describe('debug logging', function() {
    var testPacket1 = {type: 'test', data: {}, id: 0}
    var testPacket2 = {type: 'test', data: {hello: 'world'}}

    beforeEach(function() {
      socket.connect('https://heimhost/test', 'ezzie')
      sinon.stub(console, 'log')
      socket._logPackets = true
    })

    afterEach(function() {
      console.log.restore()
    })

    it('should output packets sent', function() {
      socket.send(testPacket1)
      sinon.assert.calledWithExactly(console.log, testPacket1)
    })

    it('should output packets received', function() {
      socket._onMessage({data: JSON.stringify(testPacket2)})
      sinon.assert.calledWithExactly(console.log, testPacket2)
    })

    it('should output packets sent and response received when sent with log flag', function() {
      socket._logPackets = false

      socket.send(testPacket1, true)
      sinon.assert.calledWithExactly(console.log, testPacket1)

      console.log.reset()

      socket._onMessage({data: JSON.stringify(testPacket1)})
      sinon.assert.calledWithExactly(console.log, testPacket1)
    })
  })
})
