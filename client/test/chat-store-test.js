import support from './support/setup'
import _ from 'lodash'
import assert from 'assert'
import sinon from 'sinon'
import Immutable from 'immutable'

import chat from '../lib/stores/chat'
import storage from '../lib/stores/storage'


describe('chat store', () => {
  let clock

  const startTime = chat.store.seenTTL + 100 * 1000
  support.fakeEnv({
    HEIM_ORIGIN: 'https://heimhost',
    HEIM_PREFIX: '/test',
  })

  beforeEach(() => {
    clock = support.setupClock()
    clock.tick(startTime)
    sinon.stub(chat.actions, 'messageReceived')
    sinon.stub(chat.actions, 'messagesChanged')
    sinon.stub(storage, 'setRoom')
    sinon.stub(console, 'warn')
    support.resetStore(chat.store)
    chat.store.socket = {
      on: sinon.spy(),
      endBuffering: sinon.spy(),
      send: sinon.spy(),
      pingIfIdle: sinon.spy(),
    }
    window.Raven = {setUserContext: sinon.stub()}
  })

  afterEach(() => {
    clock.restore()
    chat.actions.messageReceived.restore()
    chat.actions.messagesChanged.restore()
    storage.setRoom.restore()
    console.warn.restore()  // eslint-disable-line no-console
    window.Raven = null
  })

  function handleSocket(ev, callback) {
    // FIXME: ev data needs to be cloned when used by chat unit tests,
    // since socket events are mutated by the processing code.
    support.listenOnce(chat.store, callback)
    chat.store.socketEvent(ev)
  }

  const helloEvent = {
    'type': 'hello-event',
    'data': {
      'session': {
        'id': 'agent:tester1',
        'is_manager': true,
        'is_staff': false,
      },
    },
  }

  const message1 = {
    'id': 'id1',
    'time': startTime / 1000 - 2,
    'sender': {
      'session_id': '32.64.96.128:12345',
      'id': 'agent:tester1',
      'name': 'tester',
    },
    'content': 'test',
  }

  const message2 = {
    'id': 'id2',
    'time': startTime / 1000 - 1,
    'sender': {
      'session_id': '32.64.96.128:12345',
      'id': 'agent:tester1',
      'name': 'tester',
    },
    'content': 'test2',
  }

  const message3 = {
    'id': 'id3',
    'parent': 'id2',
    'time': startTime / 1000,
    'sender': {
      'session_id': '32.64.96.128:12346',
      'id': 'agent:tester2',
      'name': 'tester2',
    },
    'content': 'test3',
  }

  const logReply = {
    'id': '0',
    'type': 'log-reply',
    'data': {
      'log': [
        message1,
        message2,
        message3,
      ],
    },
  }

  const message0 = {
    'id': 'id0',
    'time': 0,
    'sender': {
      'session_id': '32.64.96.128:12345',
      'id': 'agent:tester1',
      'name': 'tester',
    },
    'content': 'test',
  }

  const moreLogReply = {
    'id': '0',
    'type': 'log-reply',
    'data': {
      'log': [
        message0,
      ],
      'before': 'id1',
    },
  }

  const whoReply = {
    'id': '0',
    'type': 'who-reply',
    'data': {
      'listing': [
        {
          'session_id': '32.64.96.128:12344',
          'id': 'agent:tester1',
          'name': '000tester',
          'server_id': '1a2a3a4a5a6a',
          'server_era': '1b2b3b4b5b6b',
        },
        {
          'session_id': '32.64.96.128:12345',
          'id': 'agent:tester1',
          'name': 'guest',
          'server_id': '1a2a3a4a5a6a',
          'server_era': '1b2b3b4b5b6b',
        },
        {
          'session_id': '32.64.96.128:12346',
          'id': 'agent:tester2',
          'name': 'tester2',
          'server_id': '1x2x3x4x5x6x',
          'server_era': '1y2y3y4y5y6y',
        },
      ],
    },
  }

  const nickReply = {
    'id': '1',
    'type': 'nick-reply',
    'data': {
      'session_id': '32.64.96.128:12345',
      'id': 'agent:tester1',
      'from': 'guest',
      'to': 'tester',
    },
  }

  const snapshotReply = {
    'id': '',
    'type': 'snapshot-event',
    'data': {
      'version': 'deadbeef',
      'identity': 'agent:tester1',
      'session_id': 'aabbccddeeff0011-00000abc',
      'listing': whoReply.data.listing,
      'log': logReply.data.log,
    },
  }

  const bounceEvent = {
    'id': '1',
    'type': 'bounce-event',
    'data': {
      'reason': 'authentication required',
      'auth_options': null,
    },
  }

  const successfulAuthReplyEvent = {
    'id': '1',
    'type': 'auth-reply',
    'data': {
      'success': true,
    },
  }

  const mockStorage = {
    room: {
      ezzie: {
        nick: 'tester',
        auth: {
          type: 'passcode',
          data: 'hunter2',
        },
      },
    },
  }

  const mockActivity = {
    lastActive: {
      ezzie: startTime - 20 * 1000,
    },
    lastVisit: {
      ezzie: startTime - 60 * 1000,
    },
  }

  function testErrorLogging(type, error, done) {
    const errorEvent = {
      'type': type,
      'error': error,
    }
    handleSocket(errorEvent, () => {
      sinon.assert.calledOnce(console.warn)  // eslint-disable-line no-console
      sinon.assert.calledWithExactly(console.warn, sinon.match.string, errorEvent.error)  // eslint-disable-line no-console
      done()
    })
  }

  it('should initialize with null connected and false joined state', () => {
    assert.equal(chat.store.getInitialState().connected, null)
    assert.equal(chat.store.getInitialState().joined, false)
  })

  it('should initialize with empty collections', () => {
    const initialState = chat.store.getInitialState()
    assert.equal(initialState.messages.size, 0)
    assert.equal(initialState.who.size, 0)
    assert.deepEqual(initialState.nickHues, {})
    assert(Immutable.is(initialState.roomSettings, Immutable.Map()))
  })

  describe('setup action', () => {
    beforeEach(() => {
      sinon.stub(storage, 'load')
    })

    afterEach(() => {
      storage.load.restore()
    })

    it('should save room name', done => {
      support.listenOnce(chat.store, state => {
        assert.equal(state.roomName, 'ezzie')
        done()
      })

      chat.store.setup('ezzie')
    })

    it('should load storage', () => {
      chat.store.setup('ezzie')
      sinon.assert.calledOnce(storage.load)
    })
  })

  describe('connect action', () => {
    beforeEach(() => {
      chat.store.setup('ezzie')
    })

    it('should register event handlers', () => {
      chat.store.connect()
      sinon.assert.calledWithExactly(chat.store.socket.on, 'open', chat.store.socketOpen)
      sinon.assert.calledWithExactly(chat.store.socket.on, 'close', chat.store.socketClose)
      sinon.assert.calledWithExactly(chat.store.socket.on, 'receive', chat.store.socketEvent)
    })

    it('should end socket buffering', () => {
      chat.store.connect()
      sinon.assert.calledOnce(chat.store.socket.endBuffering)
    })

    describe('then setNick action', () => {
      const testNick = 'test-nick'

      beforeEach(() => {
        chat.store.setup('ezzie')
        chat.store.connect()
        chat.store.setNick(testNick)
      })

      it('should send a nick change', () => {
        assert.equal(chat.store.state.tentativeNick, testNick)
        sinon.assert.calledWithExactly(chat.store.socket.send, {
          type: 'nick',
          data: {name: testNick},
        })
      })

      it('should avoid re-sending same nick', () => {
        chat.store.storageChange({room: {ezzie: {nick: testNick}}})
        chat.store.setNick(testNick)
        assert(chat.store.socket.send.calledOnce)
      })
    })
  })

  describe('markMessagesSeen action', () => {
    it('should store messages marked as seen, culling messages seen earlier than the TTL', () => {
      const mockSeenMessages = {
        'id1': Date.now() - chat.store.seenTTL,
        'id3': Date.now() - chat.store.seenTTL - 1,
      }
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange({room: {ezzie: {seenMessages: mockSeenMessages}}})
      chat.store.socketEvent(logReply)
      chat.store.markMessagesSeen(['id2'])
      sinon.assert.calledOnce(storage.setRoom)
      const expectedSeenMessages = {
        'id1': mockSeenMessages.id1,
        'id2': Date.now(),
      }
      sinon.assert.calledWithExactly(storage.setRoom, 'ezzie', 'seenMessages', expectedSeenMessages)
    })

    it('should not update the store if seen messages unchanged', () => {
      const mockSeenMessages = {
        'id3': Date.now(),
      }
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange({room: {ezzie: {seenMessages: mockSeenMessages}}})
      chat.store.socketEvent(logReply)
      chat.store.markMessagesSeen(['id3'])
      sinon.assert.notCalled(storage.setRoom)
    })
  })

  describe('sendMessage action', () => {
    it('should send a message', () => {
      const testContent = 'hello, ezzie!'
      chat.store.sendMessage(testContent)
      sinon.assert.calledWithExactly(chat.store.socket.send, {
        type: 'send',
        data: {content: testContent, parent: null},
      })
    })

    it('should send a message with a parent', () => {
      chat.store.socketEvent(logReply)
      const testContent = 'hello, ezzie!'
      chat.store.sendMessage(testContent, 'id1')
      sinon.assert.calledWithExactly(chat.store.socket.send, {
        type: 'send',
        data: {content: testContent, parent: 'id1'},
      })
    })
  })

  describe('when connected', () => {
    it('should have connected state: true', () => {
      support.listenOnce(chat.store, state => {
        assert.equal(state.connected, true)
      })
      chat.store.socketOpen()
    })

    it('should send stored passcode authenticaton', done => {
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange(mockStorage)
      support.listenOnce(chat.store, () => {
        assert.equal(chat.store.state.authState, 'trying-stored')
        sinon.assert.calledOnce(chat.store.socket.send)
        sinon.assert.calledWithExactly(chat.store.socket.send, {
          type: 'auth',
          data: {
            type: 'passcode',
            passcode: 'hunter2',
          },
        })
        done()
      })
      chat.store.socketOpen()
    })
  })

  describe('when disconnected', () => {
    it('should have connected state: false', () => {
      support.listenOnce(chat.store, state => {
        assert.equal(state.connected, false)
      })
      chat.store.socketClose()
    })

    it('should set joined and canJoin state to false', done => {
      support.listenOnce(chat.store, state => {
        assert.equal(state.joined, false)
        assert.equal(state.canJoin, false)
        done()
      })
      chat.store.socketClose()
    })
  })

  describe('when reconnecting', () => {
    beforeEach(() => {
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange(mockStorage)
      chat.store.joinRoom()
      chat.store.socketOpen()
      chat.store.socketEvent(successfulAuthReplyEvent)
      chat.store.socketEvent(snapshotReply)
      chat.store.socketEvent(nickReply)
      chat.store.socketClose()
      chat.store.socket.send.reset()
    })

    it('should send stored nick', done => {
      chat.store.socketOpen()
      handleSocket(snapshotReply, () => {
        sinon.assert.calledWithExactly(chat.store.socket.send, {
          type: 'nick',
          data: {name: mockStorage.room.ezzie.nick},
        })
        done()
      })
    })

    it('should send stored passcode authentication', done => {
      support.listenOnce(chat.store, () => {
        sinon.assert.calledOnce(chat.store.socket.send)
        sinon.assert.calledWithExactly(chat.store.socket.send, {
          type: 'auth',
          data: {
            type: 'passcode',
            passcode: 'hunter2',
          },
        })
        done()
      })
      chat.store.socketOpen()
    })

    it('should persist lastVisit node', done => {
      const prevLastVisit = chat.store.state.messages.get('__lastVisit')
      chat.store.socketOpen()
      handleSocket(snapshotReply, state => {
        assert(Immutable.is(state.messages.get('__lastVisit'), prevLastVisit))
        done()
      })
    })

    it('should persist shadow nodes (underscored properties)', done => {
      chat.store.state.messages.add({id: 'test', parent: 'id1', _data: 'retained'})
      assert.equal(chat.store.state.messages.get('test').get('parent'), 'id1')

      chat.store.socketOpen()
      handleSocket(snapshotReply, state => {
        const testNode = state.messages.get('test')
        assert(testNode)
        assert.equal(testNode.get('parent'), null)
        assert.equal(testNode.get('_data'), 'retained')
        done()
      })
    })
  })

  describe('on storage change', () => {
    beforeEach(() => {
      chat.store.state.roomName = 'ezzie'
    })

    it('should update auth state', () => {
      chat.store.state.connected = true
      chat.store.storageChange(mockStorage)
      assert.equal(chat.store.state.authType, 'passcode')
      assert.equal(chat.store.state.authData, 'hunter2')
    })

    it('should set tentative nick if no current nick', () => {
      assert.equal(chat.store.state.nick, null)
      chat.store.storageChange(mockStorage)
      assert.equal(chat.store.state.tentativeNick, 'tester')
    })

    it('should not set tentative nick if current nick', () => {
      chat.store.state.nick = 'test'
      chat.store.state.tentativeNick = 'unchanged'
      chat.store.storageChange(mockStorage)
      assert.equal(chat.store.state.tentativeNick, 'unchanged')
    })
  })

  describe('on activity change', () => {
    beforeEach(() => {
      chat.store.state.roomName = 'ezzie'
    })

    it('should create last visit tree node', () => {
      chat.store.activityChange(mockActivity)
      const lastVisitNode = chat.store.state.messages.get('__lastVisit')
      assert(lastVisitNode)
      assert.equal(lastVisitNode.get('time'), mockActivity.lastVisit.ezzie / 1000)
    })
  })

  describe('when ui becomes active', () => {
    describe('when connected', () => {
      it('should ping the server', () => {
        chat.store.state.connected = true
        chat.store.onActive()
        sinon.assert.calledOnce(chat.store.socket.pingIfIdle)
      })
    })

    describe('when disconnected', () => {
      it('should do nothing', () => {
        chat.store.state.connected = false
        chat.store.onActive()
        sinon.assert.notCalled(chat.store.socket.pingIfIdle)
      })
    })
  })

  describe('received hello events', () => {
    it('should store user id, manager status, and staff status', done => {
      handleSocket(helloEvent, state => {
        assert.equal(state.id, helloEvent.data.session.id)
        assert.equal(state.isManager, helloEvent.data.session.is_manager)
        assert.equal(state.isStaff, helloEvent.data.session.is_staff)
        done()
      })
    })

    it('should set auth type state to public if room not private', done => {
      const publicHelloEvent = _.merge({}, helloEvent, {data: {room_is_private: false}})
      handleSocket(publicHelloEvent, state => {
        assert.equal(state.authType, 'public')
        done()
      })
    })

    it('should set auth type state to passcode if room private', done => {
      const privateHelloEvent = _.merge({}, helloEvent, {data: {room_is_private: true}})
      handleSocket(privateHelloEvent, state => {
        assert.equal(state.authType, 'passcode')
        done()
      })
    })
  })

  describe('received messages', () => {
    const sendEvent = {
      'id': '0',
      'type': 'send-event',
      'data': message2,
    }

    const sendReplyEvent = {
      'id': '1',
      'type': 'send-event',
      'data': message3,
    }

    const sendMentionEvent = {
      'id': '2',
      'type': 'send-event',
      'data': {
        'id': 'id3',
        'time': 123456,
        'sender': {
          'session_id': '32.64.96.128:12346',
          'id': 'agent:tester2',
          'name': 'tester2',
        },
        'content': 'hey @tester',
      },
    }

    const pastSendEvent = {
      'id': '2',
      'type': 'send-event',
      'data': message1,
    }

    it('should be appended to log', done => {
      handleSocket(sendEvent, state => {
        assert(state.messages.last().isSuperset(Immutable.fromJS(sendEvent.data)))
        done()
      })
    })

    it('should be assigned a hue', done => {
      handleSocket(sendEvent, state => {
        assert.equal(state.messages.last().getIn(['sender', 'hue']), 70)
        done()
      })
    })

    it('should update sender lastSent', done => {
      handleSocket(sendEvent, state => {
        assert.equal(state.who.get(sendEvent.data.sender.session_id).get('lastSent'), sendEvent.data.time)
        done()
      })
    })

    it('should be stored as children of parent', done => {
      handleSocket(sendEvent, () => {
        handleSocket(sendReplyEvent, state => {
          assert(state.messages.get('id2').get('children').contains('id3'))
          done()
        })
      })
    })

    it('should be sorted by timestamp', done => {
      handleSocket(sendEvent, () => {
        handleSocket(pastSendEvent, state => {
          assert.deepEqual(state.messages.get('__root').get('children').toJS(), ['id1', 'id2'])
          done()
        })
      })
    })

    it('should trigger messageReceived action', done => {
      handleSocket(sendEvent, state => {
        sinon.assert.calledOnce(chat.actions.messageReceived)
        sinon.assert.calledWithExactly(chat.actions.messageReceived, state.messages.last(), state)
        done()
      })
    })

    it('should trigger messagesChanged action', done => {
      handleSocket(sendEvent, state => {
        sinon.assert.calledOnce(chat.actions.messagesChanged)
        sinon.assert.calledWithExactly(chat.actions.messagesChanged, ['__root', 'id2'], state)
        done()
      })
    })

    it('should be tagged as a mention, if it matches', done => {
      chat.store.state.tentativeNick = 'test er'
      handleSocket(sendMentionEvent, state => {
        assert(state.messages.last().get('_mention'))
        done()
      })
    })

    it('older than seenTTL should be marked seen = true', done => {
      const msgTime = (Date.now() - chat.store.seenTTL) / 1000 - 10
      const oldSendEvent = _.merge({}, sendEvent, {data: {time: msgTime}})
      handleSocket(oldSendEvent, state => {
        assert.equal(state.messages.last().get('_seen'), true)
        done()
      })
    })

    it('newer than seenTTL should be looked up whether seen', done => {
      const msgTime = Date.now() / 1000 - 5
      const seenSendEvent = _.merge({}, sendEvent, {data: {time: msgTime}})
      const mockSeenMessages = {}
      const seenTime = mockSeenMessages[seenSendEvent.data.id] = msgTime * 1000
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange({room: {ezzie: {seenMessages: mockSeenMessages}}})
      handleSocket(seenSendEvent, state => {
        assert.equal(state.messages.last().get('_seen'), seenTime)
        done()
      })
    })
  })

  function assertMessagesHaveHues(messages) {
    assert(messages.mapDFS((message, children, depth) => {
      const childrenOk = children.every(_.identity)
      return childrenOk && (depth === 0 || message.hasIn(['sender', 'hue']))
    }))
  }

  function checkLogs(msgBody) {
    it('messages should be assigned to log', done => {
      handleSocket(msgBody, state => {
        assert.equal(state.messages.size, logReply.data.log.length)
        assert(state.messages.get('id1').isSuperset(Immutable.fromJS(message1)))
        assert(state.messages.get('id2').isSuperset(Immutable.fromJS(message2)))
        assert(state.messages.get('id3').isSuperset(Immutable.fromJS(message3)))
        assert(state.messages.get('id2').get('children').contains('id3'))
        done()
      })
    })

    it('messages should all be assigned hues', done => {
      handleSocket(msgBody, state => {
        assertMessagesHaveHues(state.messages)
        done()
      })
    })

    it('messages should update sender lastSent', done => {
      handleSocket(msgBody, state => {
        assert.equal(state.who.get(message2.sender.session_id).get('lastSent'), message2.time)
        assert.equal(state.who.get(message3.sender.session_id).get('lastSent'), message3.time)
        done()
      })
    })

    it('should update earliestLog', done => {
      handleSocket(msgBody, state => {
        assert.equal(state.earliestLog, 'id1')
        done()
      })
    })
  }

  describe('sending messages', () => {
    it('should log a warning upon error', done => {
      testErrorLogging('send-reply', 'bzzt!', done)
    })
  })

  function testEditMessageEvent(type) {
    const deleteEvent = {
      'id': '0',
      'type': type,
      'data': _.merge({}, message1, {deleted: 12345}),
    }

    it('should update the message data in the tree', done => {
      chat.store.socketEvent(logReply)
      handleSocket(deleteEvent, state => {
        assert(state.messages.get(message1.id).get('deleted') === 12345)
        done()
      })
    })
  }

  describe('received edit-message-event events', () => {
    testEditMessageEvent('edit-message-event')
  })

  describe('received edit-message-reply events', () => {
    testEditMessageEvent('edit-message-reply')

    it('should log a warning upon error', done => {
      testErrorLogging('edit-message-reply', 'oh no!', done)
    })
  })

  describe('received ban-reply events', () => {
    const banReplyEvent = {
      'id': '0',
      'type': 'ban-reply',
      'data': {
        'id': 'agent:tester2',
        'seconds': 60 * 60,
      },
    }

    it('should add the id to the banned ids set', done => {
      handleSocket(banReplyEvent, state => {
        assert(state.bannedIds.has(banReplyEvent.data.id))
        done()
      })
    })

    it('should log a warning upon error', done => {
      testErrorLogging('ban-reply', 'oops!', done)
    })
  })

  function checkMessagesChangedEvent(msgBody) {
    it('should trigger messagesChanged action', done => {
      chat.actions.messagesChanged.reset()
      handleSocket(msgBody, state => {
        const ids = Immutable.Seq(msgBody.data.log).map(msg => msg.id).toArray()
        ids.unshift('__root')
        sinon.assert.calledOnce(chat.actions.messagesChanged)
        sinon.assert.calledWithExactly(chat.actions.messagesChanged, ids, state)
        done()
      })
    })
  }

  describe('received logs', () => {
    checkLogs(logReply)
    checkMessagesChangedEvent(logReply)

    it('should ignore empty logs', done => {
      const emptyLogReply = {
        'id': '0',
        'type': 'log-reply',
        'data': {
          'log': [],
        },
      }

      handleSocket(logReply, () => {
        handleSocket(emptyLogReply, state => {
          assert.equal(state.messages.size, 3)
          done()
        })
      })
    })

    describe('receiving more logs', () => {
      it('messages should be added to logs', done => {
        handleSocket(logReply, () => {
          handleSocket(moreLogReply, state => {
            assert.equal(state.messages.size, logReply.data.log.length + 1)
            assert(state.messages.get('id0').isSuperset(Immutable.fromJS(message0)))
            done()
          })
        })
      })

      it('messages should all be assigned hues', done => {
        handleSocket(logReply, () => {
          handleSocket(moreLogReply, state => {
            assertMessagesHaveHues(state.messages)
            done()
          })
        })
      })

      it('messages should update sender lastSent', done => {
        handleSocket(logReply, () => {
          handleSocket(moreLogReply, state => {
            assert.equal(state.who.get(message0.sender.session_id).get('lastSent'), message0.time)
            done()
          })
        })
      })

      it('should update earliestLog', done => {
        handleSocket(logReply, () => {
          handleSocket(moreLogReply, state => {
            assert.equal(state.earliestLog, 'id0')
            done()
          })
        })
      })
    })

    describe('receiving redundant logs', () => {
      beforeEach(() => {
        chat.store.socketEvent(logReply)
      })

      describe('should not change', () => {
        checkLogs(logReply)
      })

      it('should not trigger messagesChanged action', done => {
        const logReplyWithBefore = _.merge(_.clone(logReply), {data: {before: 'id0'}})
        chat.actions.messagesChanged.reset()
        handleSocket(logReplyWithBefore, () => {
          sinon.assert.notCalled(chat.actions.messagesChanged)
          done()
        })
      })
    })

    describe('loadMoreLogs action', () => {
      it('should not make a request if initial logs not loaded yet', () => {
        chat.store.loadMoreLogs()
        sinon.assert.notCalled(chat.store.socket.send)
      })

      it('should request 50 more logs before the earliest message', () => {
        chat.store.socketEvent(logReply)
        chat.store.loadMoreLogs()
        sinon.assert.calledWithExactly(chat.store.socket.send, {
          type: 'log',
          data: {n: 50, before: 'id1'},
        })
      })

      it('should not make a request if one already in flight', done => {
        chat.store.socketEvent(logReply)
        chat.store.loadMoreLogs()
        chat.store.loadMoreLogs()
        sinon.assert.calledOnce(chat.store.socket.send)
        handleSocket(moreLogReply, () => {
          chat.store.loadMoreLogs()
          sinon.assert.calledTwice(chat.store.socket.send)
          done()
        })
      })

      describe('status indicator', () => {
        beforeEach(() => {
          chat.store.socketEvent(logReply)
        })

        it('should be set when loading more logs', () => {
          chat.store.loadMoreLogs()
          assert.equal(chat.store.state.loadingLogs, true)
        })

        it('should be reset 250ms after no more logs received', () => {
          chat.store.loadMoreLogs()
          assert.equal(chat.store.state.loadingLogs, true)
          clock.tick(1000)
          chat.store.socketEvent(moreLogReply)
          assert.equal(chat.store.state.loadingLogs, true)
          clock.tick(100)
          assert.equal(chat.store.state.loadingLogs, true)
          clock.tick(150)
          assert.equal(chat.store.state.loadingLogs, false)
        })
      })
    })
  })

  function checkUsers(msgBody) {
    it('users should be assigned to user list', done => {
      handleSocket(msgBody, state => {
        assert.equal(state.who.size, whoReply.data.listing.length)
        assert(Immutable.Iterable(whoReply.data.listing).every(user => {
          const whoEntry = state.who.get(user.session_id)
          return !!whoEntry && whoEntry.isSuperset(Immutable.fromJS(user))
        }))
        done()
      })
    })

    it('users should all be assigned hues', done => {
      handleSocket(msgBody, state => {
        assert(state.who.every(whoEntry => {
          return !!whoEntry.has('hue')
        }))
        done()
      })
    })
  }

  describe('received users', () => {
    checkUsers(whoReply)
  })

  describe('received snapshots', () => {
    checkLogs(snapshotReply)
    checkMessagesChangedEvent(snapshotReply)
    checkUsers(snapshotReply)

    it('should update server version', done => {
      handleSocket(snapshotReply, state => {
        assert.equal(state.serverVersion, snapshotReply.data.version)
        done()
      })
    })

    it('should update session id', done => {
      handleSocket(snapshotReply, state => {
        assert.equal(state.sessionId, snapshotReply.data.session_id)
        done()
      })
    })

    it('should set canJoin state to true', done => {
      handleSocket(snapshotReply, state => {
        assert.equal(state.canJoin, true)
        done()
      })
    })

    describe('on join', () => {
      beforeEach(() => {
        chat.store.joinRoom()
      })

      it('should set joined state to the join time', done => {
        handleSocket(snapshotReply, state => {
          assert.equal(state.joined, Date.now())
          done()
        })
      })

      it('should clear auth state', done => {
        chat.store.state.authState = 'trying-stored'
        handleSocket(snapshotReply, state => {
          assert.equal(state.authState, null)
          done()
        })
      })

      it('should trigger sending stored nick', done => {
        chat.store.state.roomName = 'ezzie'
        chat.store.storageChange(mockStorage)
        handleSocket(snapshotReply, () => {
          sinon.assert.calledWithExactly(chat.store.socket.send, {
            type: 'nick',
            data: {name: mockStorage.room.ezzie.nick},
          })
          done()
        })
      })

      it('should not send stored nick if unset', done => {
        chat.store.state.roomName = 'ezzie'
        chat.store.storageChange({room: {}})
        handleSocket(snapshotReply, () => {
          sinon.assert.notCalled(chat.store.socket.send)
          done()
        })
      })
    })
  })

  describe('received nick changes', () => {
    const rejectedNickReply = {
      'id': '1',
      'type': 'nick-reply',
      'error': 'error',
    }

    const nonexistentNickEvent = {
      'id': '2',
      'type': 'nick-event',
      'data': {
        'session_id': '32.64.96.128:54321',
        'id': 'agent:noman',
        'from': 'nonexistence',
        'to': 'absence',
      },
    }

    beforeEach(() => {
      chat.store.socketEvent(helloEvent)
      chat.store.socketEvent(snapshotReply)
    })

    it('should update user list name', done => {
      handleSocket(whoReply, () => {
        handleSocket(nickReply, state => {
          assert.equal(state.who.getIn([nickReply.data.session_id, 'name']), nickReply.data.to)
          done()
        })
      })
    })

    it('should update hue', done => {
      handleSocket(whoReply, () => {
        handleSocket(nickReply, state => {
          assert.equal(state.who.getIn([nickReply.data.session_id, 'hue']), 70)
          done()
        })
      })
    })

    it('should add nonexistent users', done => {
      handleSocket(whoReply, () => {
        handleSocket(nonexistentNickEvent, state => {
          assert(state.who.has(nonexistentNickEvent.data.session_id))
          done()
        })
      })
    })

    describe('in response to nick set', () => {
      it('should not update nick if rejected', done => {
        chat.store.state.nick = 'previous'
        chat.store.state.roomName = 'ezzie'
        handleSocket(rejectedNickReply, state => {
          assert.equal(state.nick, 'previous')
          done()
        })
      })

      it('should update stored nick', done => {
        chat.store.state.roomName = 'ezzie'
        handleSocket(nickReply, state => {
          assert.equal(state.nick, 'tester')
          sinon.assert.calledOnce(storage.setRoom)
          sinon.assert.calledWithExactly(storage.setRoom, 'ezzie', 'nick', 'tester')
          done()
        })
      })

      it('should update Raven user context', done => {
        handleSocket(nickReply, () => {
          sinon.assert.calledOnce(Raven.setUserContext)
          sinon.assert.calledWithExactly(Raven.setUserContext, {
            'id': 'agent:tester1',
            'nick': 'tester',
            'session_id': 'aabbccddeeff0011-00000abc',
          })
          done()
        })
      })
    })
  })

  describe('received join events', () => {
    const joinEvent = {
      'id': '1',
      'type': 'join-event',
      'data': {
        'session_id': '32.64.96.128:12347',
        'id': 'agent:someone',
        'name': '32.64.96.128:12347',
        'server_id': '1a2a3a4a5a6a',
        'server_era': '1b2b3b4b5b6b',
      },
    }

    it('should add to user list', done => {
      handleSocket(joinEvent, state => {
        assert(state.who.get(joinEvent.data.session_id).isSuperset(Immutable.fromJS(joinEvent.data)))
        done()
      })
    })

    it('should assign a hue', done => {
      handleSocket(joinEvent, state => {
        assert.equal(state.who.getIn([joinEvent.data.session_id, 'hue']), 50)
        done()
      })
    })
  })

  describe('received part events', () => {
    const partEvent = {
      'id': '1',
      'type': 'part-event',
      'data': {
        'session_id': '32.64.96.128:12345',
        'id': 'agent:tester1',
        'name': 'tester',
      },
    }

    it('should remove from user list', done => {
      handleSocket(whoReply, () => {
        handleSocket(partEvent, state => {
          assert(!state.who.has(partEvent.data.session_id))
          done()
        })
      })
    })
  })

  describe('setRoomSettings action', () => {
    it('should merge with roomSettings data', () => {
      chat.store.setRoomSettings({testing: true, another: {test: false}})
      assert.equal(chat.store.state.roomSettings.get('testing'), true)
      assert.equal(chat.store.state.roomSettings.getIn(['another', 'test']), false)
      chat.store.setRoomSettings({another: {test: true}})
      assert.equal(chat.store.state.roomSettings.getIn(['another', 'test']), true)
    })
  })

  describe('tryRoomPasscode action', () => {
    it('should set authData and send an auth attempt', () => {
      const testPassword = 'hunter2'
      chat.store.tryRoomPasscode(testPassword)
      assert.equal(chat.store.state.authData, testPassword)
      assert.equal(chat.store.state.authState, 'trying')
      sinon.assert.calledOnce(chat.store.socket.send)
      sinon.assert.calledWithExactly(chat.store.socket.send, {
        type: 'auth',
        data: {
          type: 'passcode',
          passcode: testPassword,
        },
      })
    })
  })

  describe('received bounce events', () => {
    it('should set passcode auth', done => {
      handleSocket(bounceEvent, state => {
        assert.equal(state.authType, 'passcode')
        done()
      })
    })

    it('should set canJoin state to false', done => {
      handleSocket(bounceEvent, state => {
        assert.equal(state.canJoin, false)
        done()
      })
    })

    describe('if not trying a stored passcode', () => {
      it('should set auth state to "needs-passcode"', done => {
        handleSocket(bounceEvent, state => {
          assert.equal(state.authState, 'needs-passcode')
          done()
        })
      })
    })

    describe('if trying a stored passcode', () => {
      it('should be ignored', done => {
        chat.store.state.authState = 'trying-stored'
        handleSocket(bounceEvent, state => {
          assert.equal(state.authState, 'trying-stored')
          done()
        })
      })
    })
  })

  describe('received auth reply events', () => {
    const incorrectAuthReplyEvent = {
      'id': '1',
      'type': 'auth-reply',
      'data': {
        'success': false,
        'reason': 'passcode incorrect',
      },
    }

    const errorAuthReplyEvent = {
      'id': '1',
      'type': 'auth-reply',
      'data': null,
      'error': 'command not implemented',
    }

    const redundantAuthReplyEvent = {
      'id': '1',
      'type': 'auth-reply',
      'data': null,
      'error': 'already joined',
    }

    beforeEach(() => {
      chat.store.state.roomName = 'ezzie'
      chat.store.state.authType = 'passcode'
      chat.store.state.authData = 'hunter2'
    })

    describe('if successful', () => {
      it('should save auth data in storage', done => {
        handleSocket(successfulAuthReplyEvent, state => {
          sinon.assert.calledOnce(storage.setRoom)
          sinon.assert.calledWithExactly(storage.setRoom, 'ezzie', 'auth', {type: 'passcode', data: 'hunter2'})
          assert.equal(state.authState, null)
          done()
        })
      })
    })

    function testAuthFail(body) {
      describe('if stored auth unsuccessful', () => {
        it('should set auth state to "needs-passcode"', () => {
          chat.store.state.authState = 'trying-stored'
          handleSocket(body, state => {
            assert.equal(state.authState, 'needs-passcode')
          })
        })
      })

      describe('if auth unsuccessful', () => {
        it('should set auth state to "failed"', () => {
          chat.store.state.authState = 'trying'
          handleSocket(body, state => {
            assert.equal(state.authState, 'failed')
          })
        })
      })
    }

    describe('in case of error', () => {
      testAuthFail(errorAuthReplyEvent)
    })

    testAuthFail(incorrectAuthReplyEvent)

    describe('if auth redundant', () => {
      it('should not change auth state', () => {
        chat.store.state.authState = null
        handleSocket(redundantAuthReplyEvent, state => {
          assert.equal(state.authState, null)
        })
      })
    })
  })

  describe('received network partition events', () => {
    const networkPartitionEvent = {
      'id': '1',
      'type': 'network-event',
      'data': {
        'type': 'partition',
        'server_id': '1a2a3a4a5a6a',
        'server_era': '1b2b3b4b5b6b',
      },
    }

    it('should remove all associated users from the user list', done => {
      handleSocket(whoReply, () => {
        handleSocket(networkPartitionEvent, state => {
          assert.equal(state.who.size, 1)
          assert.equal(state.who.first().get('id'), whoReply.data.listing[2].id)
          done()
        })
      })
    })
  })

  describe('received ping events', () => {
    it('should be ignored', () => {
      const storeSpy = sinon.spy()
      support.listenOnce(chat.store, storeSpy)
      chat.store.socketEvent({type: 'ping-event'})
      chat.store.socketEvent({type: 'ping-reply'})
      sinon.assert.notCalled(storeSpy)
    })
  })

  describe('received unknown chat events', () => {
    const unknownEvent = {
      'id': '1',
      'type': 'wat-event',
      'data': {
        'wat': 'wat',
      },
    }

    it('should log a warning', done => {
      handleSocket(unknownEvent, () => {
        sinon.assert.calledOnce(console.warn)  // eslint-disable-line no-console
        sinon.assert.calledWithExactly(console.warn, sinon.match.string, unknownEvent.type)  // eslint-disable-line no-console
        done()
      })
    })
  })
})
