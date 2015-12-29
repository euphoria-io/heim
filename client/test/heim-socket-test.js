import support from './support/setup'
import assert from 'assert'
import sinon from 'sinon'

import Socket from '../lib/heim/Socket'


describe('socket store', () => {
  let clock
  const realWebSocket = window.WebSocket
  let fakeWebSocket
  let fakeWebSocketContructor
  let socket

  beforeEach(() => {
    clock = support.setupClock()
    fakeWebSocketContructor = sinon.spy(function initSpy() {
      fakeWebSocket = this
      this.send = sinon.spy()
      this.close = sinon.spy()
    })
    window.WebSocket = fakeWebSocketContructor
    socket = new Socket()
  })

  afterEach(() => {
    clock.restore()
    window.WebSocket = realWebSocket
  })

  describe('_wsurl', () => {
    it('should return wss://host/room/name/ws?h=1 if protocol is https', () => {
      assert.equal(socket._wsurl('https://host/prefix', 'ezzie'), 'wss://host/prefix/room/ezzie/ws?h=1')
    })

    it('should return ws://host/room/name/ws?h=1 if protocol is NOT https', () => {
      assert.equal(socket._wsurl('http://host/prefix', 'ezzie'), 'ws://host/prefix/room/ezzie/ws?h=1')
    })
  })

  describe('buffering', () => {
    beforeEach(() => {
      sinon.stub(socket.events, 'emit')
    })

    afterEach(() => {
      socket.events.emit.restore()
    })

    it('should init with buffering off', () => {
      assert.equal(socket._buffer, null)
    })

    it('when started should suppress events and store them until end', () => {
      const receiveObj = {test: true}
      socket.startBuffering()
      socket._emit('open')
      socket._emit('receive', receiveObj)
      sinon.assert.notCalled(socket.events.emit)
      assert.deepEqual(socket._buffer, [
        ['open', undefined],
        ['receive', receiveObj],
      ])
      socket.endBuffering()
      sinon.assert.calledTwice(socket.events.emit)
      sinon.assert.calledWithExactly(socket.events.emit, 'open', undefined)
      sinon.assert.calledWithExactly(socket.events.emit, 'receive', receiveObj)
      assert.equal(socket._buffer, null)

      socket.events.emit.reset()
      socket._emit('receive', receiveObj)
      sinon.assert.calledOnce(socket.events.emit)
      sinon.assert.calledWithExactly(socket.events.emit, 'receive', receiveObj)
    })
  })

  describe('connect method', () => {
    it('should by connect to wss://heimhost/test/ws?h=1 with heim1 protocol', () => {
      socket.connect('https://heimhost/test', 'ezzie')
      const expectedPath = 'wss://heimhost/test/room/ezzie/ws?h=1'
      sinon.assert.calledWithExactly(fakeWebSocketContructor, expectedPath, 'heim1')
    })

    it('should set up event handlers', () => {
      socket.connect('https://heimhost/test', 'ezzie')
      assert(fakeWebSocket.onopen)
      assert(fakeWebSocket.onclose)
      assert(fakeWebSocket.onmessage)
    })
  })

  describe('when socket opened', () => {
    it('should emit an open event', done => {
      socket.once('open', done)
      socket._onOpen()
    })
  })

  function checkSocketCleanup(action) {
    it('should emit an close event', done => {
      socket.once('close', done)
      action()
    })

    it('should clean up timeouts', () => {
      const pingTimeout = socket.pingTimeout = 1
      const pingReplyTimeout = socket.pingReplyTimeout = 2
      sinon.stub(window, 'clearTimeout')
      action()
      sinon.assert.calledTwice(window.clearTimeout)
      sinon.assert.calledWithExactly(window.clearTimeout, pingTimeout)
      sinon.assert.calledWithExactly(window.clearTimeout, pingReplyTimeout)
      window.clearTimeout.restore()
    })
  }

  describe('when socket closed', () => {
    beforeEach(() => {
      socket.connect('https://heimhost/test', 'ezzie')
      sinon.stub(socket, 'reconnect')
    })

    afterEach(() => {
      socket.reconnect.restore()
    })

    checkSocketCleanup(() => socket.ws.onclose())

    it('should clear socket event handlers', () => {
      socket.ws.onclose()
      assert.equal(fakeWebSocket.onopen, null)
      assert.equal(fakeWebSocket.onclose, null)
      assert.equal(fakeWebSocket.onmessage, null)
    })

    it('should attempt to reconnect within 5s', () => {
      socket.ws.onclose()
      clock.tick(5000)
      sinon.assert.calledOnce(socket.reconnect)
    })
  })

  describe('a forceful reconnect', () => {
    beforeEach(() => {
      socket.connect('https://heimhost/test', 'ezzie')
    })

    checkSocketCleanup(() => socket.reconnect())

    it('should close the socket and connect again', () => {
      const oldWs = socket.ws
      fakeWebSocketContructor.reset()
      socket.reconnect()
      sinon.assert.calledOnce(oldWs.close)
      sinon.assert.calledOnce(fakeWebSocketContructor)
    })
  })

  describe('when message received', () => {
    it('should emit a receive event', done => {
      const testBody = {it: 'works'}

      socket.once('receive', body => {
        assert.deepEqual(body, testBody)
        done()
      })

      socket._onMessage({data: JSON.stringify(testBody)})
    })
  })

  describe('when server ping received', () => {
    beforeEach(() => {
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

    afterEach(() => {
      window.setTimeout.restore()
    })

    it('should send a ping-reply', () => {
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping-reply',
        data: {
          time: 0,
        },
        id: '0',
      }))
    })

    it('should schedule timeout', () => {
      sinon.assert.calledWith(setTimeout, sinon.match.func, 20 * 1000)
    })

    describe('when a second ping received late', () => {
      beforeEach(() => {
        setTimeout.reset()
        socket._onMessage({data: JSON.stringify({
          type: 'ping-event',
          data: {
            time: 0,
            next: 10,
          },
        })})
      })

      it('should not schedule timeout', () => {
        sinon.assert.notCalled(setTimeout)
      })
    })

    describe('if another server ping isn\'t received before the next timeout', () => {
      beforeEach(() => {
        fakeWebSocket.send.reset()
        sinon.stub(socket, 'reconnect')
        clock.tick(20000)
      })

      afterEach(() => {
        socket.reconnect.restore()
        clearTimeout(socket.pingTimeout)
        clearTimeout(socket.pingReplyTimeout)
      })

      it('should send a client ping', () => {
        sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
          type: 'ping',
          id: '1',
          data: {},
        }))
      })

      describe('after 2000ms', () => {
        describe('if there is no response', () => {
          it('should force a reconnect', () => {
            clock.tick(2000)
            sinon.assert.calledOnce(socket.reconnect)
          })
        })

        describe('if any server message received', () => {
          it('should not reconnect', () => {
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

  describe('pingIfIdle action', () => {
    beforeEach(() => {
      socket.connect('https://heimhost/test', 'ezzie')
    })

    it('should send a ping if no messages have ever been received', () => {
      socket.pingIfIdle()
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'ping',
        id: '0',
        data: {},
      }))
    })

    it('should send a ping if no messages have been received in the last 2000ms', () => {
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

    it('should not send a ping if a message has been received in the last 2000ms', () => {
      socket._onMessage({data: JSON.stringify({
        type: 'hello, ezzie.',
      })})
      clock.tick(1000)
      socket.pingIfIdle()
      sinon.assert.notCalled(fakeWebSocket.send)
    })

    it('should not send a second ping if one was sent in the last 2000ms', () => {
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

  describe('send action', () => {
    beforeEach(() => {
      socket.connect('https://heimhost/test', 'ezzie')
    })

    it('should send JSON to the websocket', () => {
      socket.send({
        type: 'send',
        data: {
          content: 'hello, ezzie.',
        },
      })
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({
        type: 'send',
        data: {
          content: 'hello, ezzie.',
        },
        id: '0',
      }))
    })

    it('should increment sequence number', () => {
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

    it('should send a data property even if unset', () => {
      socket.send({})
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({id: '0', data: {}}))
    })

    it('should send an id property if specified', () => {
      socket.send({id: 'heyyy'})
      sinon.assert.calledWith(fakeWebSocket.send, JSON.stringify({id: 'heyyy', data: {}}))
    })
  })

  describe('debug logging', () => {
    const testPacket1 = {type: 'test', data: {}, id: 0}
    const testPacket2 = {type: 'test', data: {hello: 'world'}}

    beforeEach(() => {
      socket.connect('https://heimhost/test', 'ezzie')
      sinon.stub(console, 'log')  // eslint-disable-line no-console
      socket._logPackets = true
    })

    afterEach(() => {
      console.log.restore()  // eslint-disable-line no-console
    })

    it('should output packets sent', () => {
      socket.send(testPacket1)
      sinon.assert.calledWithExactly(console.log, testPacket1)  // eslint-disable-line no-console
    })

    it('should output packets received', () => {
      socket._onMessage({data: JSON.stringify(testPacket2)})
      sinon.assert.calledWithExactly(console.log, testPacket2)  // eslint-disable-line no-console
    })

    it('should output packets sent and response received when sent with log flag', () => {
      socket._logPackets = false

      socket.send(testPacket1, true)
      sinon.assert.calledWithExactly(console.log, testPacket1)  // eslint-disable-line no-console

      console.log.reset()  // eslint-disable-line no-console

      socket._onMessage({data: JSON.stringify(testPacket1)})
      sinon.assert.calledWithExactly(console.log, testPacket1)  // eslint-disable-line no-console
    })
  })
})
