var support = require('./support/setup')
var _ = require('lodash')
var assert = require('assert')
var sinon = require('sinon')
var Immutable = require('immutable')


describe('chat store', function() {
  var chat = require('../lib/stores/chat')
  var socket = require('../lib/stores/socket')
  var storage = require('../lib/stores/storage')
  var clock

  var startTime = chat.store.seenTTL + 100 * 1000

  beforeEach(function() {
    clock = support.setupClock()
    clock.tick(startTime)
    sinon.stub(chat.actions, 'messageReceived')
    sinon.stub(chat.actions, 'messagesChanged')
    sinon.stub(socket, 'send')
    sinon.stub(storage, 'setRoom')
    sinon.stub(console, 'warn')
    support.resetStore(chat.store)
    window.Raven = {setUserContext: sinon.stub()}
  })

  afterEach(function() {
    clock.restore()
    chat.actions.messageReceived.restore()
    chat.actions.messagesChanged.restore()
    socket.send.restore()
    storage.setRoom.restore()
    console.warn.restore()
    window.Raven = null
  })

  function handleSocket(ev, callback) {
    // FIXME: ev data needs to be cloned when used by chat unit tests,
    // since socket events are mutated by the processing code.
    support.listenOnce(chat.store, callback)
    chat.store.socketEvent(ev)
  }

  var helloEvent = {
    'type': 'hello-event',
    'data': {
      'id': 'agent:tester1',
      'is_manager': true,
      'is_staff': false,
    }
  }

  var message1 = {
    'id': 'id1',
    'time': startTime / 1000 - 2,
    'sender': {
      'session_id': '32.64.96.128:12345',
      'id': 'agent:tester1',
      'name': 'tester',
    },
    'content': 'test',
  }

  var message2 = {
    'id': 'id2',
    'time': startTime / 1000 - 1,
    'sender': {
      'session_id': '32.64.96.128:12345',
      'id': 'agent:tester1',
      'name': 'tester',
    },
    'content': 'test2',
  }

  var message3 = {
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

  var logReply = {
    'id': '0',
    'type': 'log-reply',
    'data': {
      'log': [
        message1,
        message2,
        message3,
      ]
    }
  }

  var message0 = {
    'id': 'id0',
    'time': 0,
    'sender': {
      'session_id': '32.64.96.128:12345',
      'id': 'agent:tester1',
      'name': 'tester',
    },
    'content': 'test',
  }

  var moreLogReply = {
    'id': '0',
    'type': 'log-reply',
    'data': {
      'log': [
        message0,
      ],
      'before': 'id1',
    }
  }

  var whoReply = {
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
      ]
    }
  }

  var nickReply = {
    'id': '1',
    'type': 'nick-reply',
    'data': {
      'session_id': '32.64.96.128:12345',
      'id': 'agent:tester1',
      'from': 'guest',
      'to': 'tester',
    }
  }

  var snapshotReply = {
    'id': '',
    'type': 'snapshot-event',
    'data': {
      'version': 'deadbeef',
      'identity': 'agent:tester1',
      'session_id': 'aabbccddeeff0011-00000abc',
      'listing': whoReply.data.listing,
      'log': logReply.data.log,
    }
  }

  var bounceEvent = {
    'id': '1',
    'type': 'bounce-event',
    'data': {
      'reason': 'authentication required',
      'auth_options': null,
    },
  }

  var successfulAuthReplyEvent = {
    'id': '1',
    'type': 'auth-reply',
    'data': {
      'success': true,
    },
  }

  var mockStorage = {
    room: {
      ezzie: {
        nick: 'tester',
        auth: {
          type: 'passcode',
          data: 'hunter2',
        },
      }
    }
  }

  var mockActivity = {
    lastActive: {
      ezzie: startTime - 20 * 1000,
    },
    lastVisit: {
      ezzie: startTime - 60 * 1000,
    },
  }

  it('should initialize with null connected and false joined state', function() {
    assert.equal(chat.store.getInitialState().connected, null)
    assert.equal(chat.store.getInitialState().joined, false)
  })

  it('should initialize with empty collections', function() {
    var initialState = chat.store.getInitialState()
    assert.equal(initialState.messages.size, 0)
    assert.equal(initialState.who.size, 0)
    assert.deepEqual(initialState.nickHues, {})
    assert(Immutable.is(initialState.roomSettings, Immutable.Map()))
  })

  describe('connect action', function() {
    beforeEach(function() {
      sinon.stub(socket, 'connect')
      sinon.stub(storage, 'load')
    })

    afterEach(function() {
      socket.connect.restore()
      storage.load.restore()
    })

    it('should connect socket with room name', function() {
      chat.store.connect('ezzie', undefined)
      sinon.assert.calledOnce(socket.connect)
      sinon.assert.calledWithExactly(socket.connect, 'ezzie', undefined)
    })

    it('should save room name', function(done) {
      support.listenOnce(chat.store, function(state) {
        assert.equal(state.roomName, 'ezzie')
        done()
      })

      chat.store.connect('ezzie')
    })

    it('should load storage', function() {
      chat.store.connect('ezzie')
      sinon.assert.calledOnce(storage.load)
    })

    describe('then setNick action', function() {
      var testNick = 'test-nick'

      beforeEach(function() {
        chat.store.connect('ezzie')
        chat.store.setNick(testNick)
      })

      it('should send a nick change', function() {
        assert.equal(chat.store.state.tentativeNick, testNick)
        sinon.assert.calledWithExactly(socket.send, {
          type: 'nick',
          data: {name: testNick},
        })
      })

      it('should avoid re-sending same nick', function() {
        chat.store.storageChange({room: {ezzie: {nick: testNick}}})
        chat.store.setNick(testNick)
        assert(socket.send.calledOnce)
      })
    })
  })

  describe('markMessagesSeen action', function() {
    it('should store messages marked as seen, culling messages seen earlier than the TTL', function() {
      var mockSeenMessages = {
        'id1': Date.now() - chat.store.seenTTL,
        'id3': Date.now() - chat.store.seenTTL - 1,
      }
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange({room: {ezzie: {seenMessages: mockSeenMessages}}})
      chat.store.socketEvent({status: 'receive', body: logReply})
      chat.store.markMessagesSeen(['id2'])
      sinon.assert.calledOnce(storage.setRoom)
      var expectedSeenMessages = {
        'id1': mockSeenMessages.id1,
        'id2': Date.now(),
      }
      sinon.assert.calledWithExactly(storage.setRoom, 'ezzie', 'seenMessages', expectedSeenMessages)
    })

    it('should not update the store if seen messages unchanged', function() {
      var mockSeenMessages = {
        'id3': Date.now(),
      }
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange({room: {ezzie: {seenMessages: mockSeenMessages}}})
      chat.store.socketEvent({status: 'receive', body: logReply})
      chat.store.markMessagesSeen(['id3'])
      sinon.assert.notCalled(storage.setRoom)
    })
  })

  describe('sendMessage action', function() {
    it('should send a message', function() {
      var testContent = 'hello, ezzie!'
      chat.store.sendMessage(testContent)
      sinon.assert.calledWithExactly(socket.send, {
        type: 'send',
        data: {content: testContent, parent: null},
      })
    })

    it('should send a message with a parent', function() {
      chat.store.socketEvent({status: 'receive', body: logReply})
      var testContent = 'hello, ezzie!'
      chat.store.sendMessage(testContent, 'id1')
      sinon.assert.calledWithExactly(socket.send, {
        type: 'send',
        data: {content: testContent, parent: 'id1'},
      })
    })
  })

  describe('when connected', function() {
    it('should have connected state: true', function() {
      handleSocket({status: 'open'}, function(state) {
        assert.equal(state.connected, true)
      })
    })

    it('should send stored passcode authenticaton', function(done) {
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange(mockStorage)
      handleSocket({status: 'open'}, function() {
        assert.equal(chat.store.state.authState, 'trying-stored')
        sinon.assert.calledOnce(socket.send)
        sinon.assert.calledWithExactly(socket.send, {
          type: 'auth',
          data: {
            type: 'passcode',
            passcode: 'hunter2',
          },
        })
        done()
      })
    })
  })

  describe('when disconnected', function() {
    it('should have connected state: false', function() {
      handleSocket({status: 'close'}, function(state) {
        assert.equal(state.connected, false)
      })
    })

    it('should set joined and canJoin state to false', function(done) {
      handleSocket({status: 'close'}, function(state) {
        assert.equal(state.joined, false)
        assert.equal(state.canJoin, false)
        done()
      })
    })
  })

  describe('when reconnecting', function() {
    beforeEach(function() {
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange(mockStorage)
      chat.store.joinRoom()
      chat.store.socketEvent({status: 'open'})
      chat.store.socketEvent({status: 'receive', body: successfulAuthReplyEvent})
      chat.store.socketEvent({status: 'receive', body: snapshotReply})
      chat.store.socketEvent({status: 'receive', body: nickReply})
      chat.store.socketEvent({status: 'close'})
      socket.send.reset()
    })

    it('should send stored nick', function(done) {
      chat.store.socketEvent({status: 'open'})
      handleSocket({status: 'receive', body: snapshotReply}, function() {
        sinon.assert.calledWithExactly(socket.send, {
          type: 'nick',
          data: {name: mockStorage.room.ezzie.nick},
        })
        done()
      })
    })

    it('should send stored passcode authentication', function(done) {
      handleSocket({status: 'open'}, function() {
        sinon.assert.calledOnce(socket.send)
        sinon.assert.calledWithExactly(socket.send, {
          type: 'auth',
          data: {
            type: 'passcode',
            passcode: 'hunter2',
          },
        })
        done()
      })
    })

    it('should persist lastVisit node', function(done) {
      var prevLastVisit = chat.store.state.messages.get('__lastVisit')
      chat.store.socketEvent({status: 'open'})
      handleSocket({status: 'receive', body: snapshotReply}, function(state) {
        assert(Immutable.is(state.messages.get('__lastVisit'), prevLastVisit))
        done()
      })
    })

    it('should persist shadow nodes (underscored properties)', function(done) {
      chat.store.state.messages.add({id: 'test', parent: 'id1', _data: 'retained'})
      assert.equal(chat.store.state.messages.get('test').get('parent'), 'id1')

      chat.store.socketEvent({status: 'open'})
      handleSocket({status: 'receive', body: snapshotReply}, function(state) {
        var testNode = state.messages.get('test')
        assert(testNode)
        assert.equal(testNode.get('parent'), null)
        assert.equal(testNode.get('_data'), 'retained')
        done()
      })
    })
  })

  describe('on storage change', function() {
    beforeEach(function() {
      chat.store.state.roomName = 'ezzie'
    })

    it('should update auth state', function() {
      chat.store.state.connected = true
      chat.store.storageChange(mockStorage)
      assert.equal(chat.store.state.authType, 'passcode')
      assert.equal(chat.store.state.authData, 'hunter2')
    })

    it('should set tentative nick if no current nick', function() {
      assert.equal(chat.store.state.nick, null)
      chat.store.storageChange(mockStorage)
      assert.equal(chat.store.state.tentativeNick, 'tester')
    })

    it('should not set tentative nick if current nick', function() {
      chat.store.state.nick = 'test'
      chat.store.state.tentativeNick = 'unchanged'
      chat.store.storageChange(mockStorage)
      assert.equal(chat.store.state.tentativeNick, 'unchanged')
    })
  })

  describe('on activity change', function() {
    beforeEach(function() {
      chat.store.state.roomName = 'ezzie'
    })

    it('should create last visit tree node', function() {
      chat.store.activityChange(mockActivity)
      var lastVisitNode = chat.store.state.messages.get('__lastVisit')
      assert(lastVisitNode)
      assert.equal(lastVisitNode.get('time'), mockActivity.lastVisit.ezzie / 1000)
    })
  })

  describe('when ui becomes active', function() {
    beforeEach(function() {
      sinon.stub(socket, 'pingIfIdle')
    })

    afterEach(function() {
      socket.pingIfIdle.restore()
    })

    describe('when connected', function() {
      it('should ping the server', function() {
        chat.store.state.connected = true
        chat.store.onActive()
        sinon.assert.calledOnce(socket.pingIfIdle)
      })
    })

    describe('when disconnected', function() {
      it('should do nothing', function() {
        chat.store.state.connected = false
        chat.store.onActive()
        sinon.assert.notCalled(socket.pingIfIdle)
      })
    })
  })

  describe('received hello events', function() {
    it('should store user id, manager status, and staff status', function(done) {
      handleSocket({status: 'receive', body: helloEvent}, function(state) {
        // jshint camelcase: false
        assert.equal(state.id, helloEvent.data.id)
        assert.equal(state.isManager, helloEvent.data.is_manager)
        assert.equal(state.isStaff, helloEvent.data.is_staff)
        done()
      })
    })
  })

  describe('received messages', function() {
    var sendEvent = {
      'id': '0',
      'type': 'send-event',
      'data': message2,
    }

    var sendReplyEvent = {
      'id': '1',
      'type': 'send-event',
      'data': message3,
    }

    var sendMentionEvent = {
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
      }
    }

    var pastSendEvent = {
      'id': '2',
      'type': 'send-event',
      'data': message1,
    }

    it('should be appended to log', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function(state) {
        assert(state.messages.last().isSuperset(Immutable.fromJS(sendEvent.data)))
        done()
      })
    })

    it('should be assigned a hue', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function(state) {
        assert.equal(state.messages.last().getIn(['sender', 'hue']), 70)
        done()
      })
    })

    it('should update sender lastSent', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function(state) {
        // jshint camelcase: false
        assert.equal(state.who.get(sendEvent.data.sender.session_id).get('lastSent'), sendEvent.data.time)
        done()
      })
    })

    it('should be stored as children of parent', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function() {
        handleSocket({status: 'receive', body: sendReplyEvent}, function(state) {
          assert(state.messages.get('id2').get('children').contains('id3'))
          done()
        })
      })
    })

    it('should be sorted by timestamp', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function() {
        handleSocket({status: 'receive', body: pastSendEvent}, function(state) {
          assert.deepEqual(state.messages.get('__root').get('children').toJS(), ['id1', 'id2'])
          done()
        })
      })
    })

    it('should trigger messageReceived action', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function(state) {
        sinon.assert.calledOnce(chat.actions.messageReceived)
        sinon.assert.calledWithExactly(chat.actions.messageReceived, state.messages.last(), state)
        done()
      })
    })

    it('should trigger messagesChanged action', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function(state) {
        sinon.assert.calledOnce(chat.actions.messagesChanged)
        sinon.assert.calledWithExactly(chat.actions.messagesChanged, ['__root', 'id2'], state)
        done()
      })
    })

    it('should be tagged as a mention, if it matches', function(done) {
      chat.store.state.tentativeNick = 'test er'
      handleSocket({status: 'receive', body: sendMentionEvent}, function(state) {
        assert(state.messages.last().get('_mention'))
        done()
      })
    })

    it('older than seenTTL should be marked seen = true', function(done) {
      var msgTime = (Date.now() - chat.store.seenTTL) / 1000 - 10
      var oldSendEvent = _.merge({}, sendEvent, {data: {time: msgTime}})
      handleSocket({status: 'receive', body: oldSendEvent}, function(state) {
        assert.equal(state.messages.last().get('_seen'), true)
        done()
      })
    })

    it('newer than seenTTL should be looked up whether seen', function(done) {
      var msgTime = Date.now() / 1000 - 5
      var seenSendEvent = _.merge({}, sendEvent, {data: {time: msgTime}})
      var mockSeenMessages = {}
      var seenTime = mockSeenMessages[seenSendEvent.data.id] = msgTime * 1000
      chat.store.state.roomName = 'ezzie'
      chat.store.storageChange({room: {ezzie: {seenMessages: mockSeenMessages}}})
      handleSocket({status: 'receive', body: seenSendEvent}, function(state) {
        assert.equal(state.messages.last().get('_seen'), seenTime)
        done()
      })
    })
  })

  function assertMessagesHaveHues(messages) {
    assert(messages.mapDFS(function(message, children, depth) {
      var childrenOk = children.every(function(v) { return v })
      return childrenOk && (depth === 0 || message.hasIn(['sender', 'hue']))
    }))
  }

  function checkLogs(msgBody) {
    it('messages should be assigned to log', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        assert.equal(state.messages.size, logReply.data.log.length)
        assert(state.messages.get('id1').isSuperset(Immutable.fromJS(message1)))
        assert(state.messages.get('id2').isSuperset(Immutable.fromJS(message2)))
        assert(state.messages.get('id3').isSuperset(Immutable.fromJS(message3)))
        assert(state.messages.get('id2').get('children').contains('id3'))
        done()
      })
    })

    it('messages should all be assigned hues', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        assertMessagesHaveHues(state.messages)
        done()
      })
    })

    it('messages should update sender lastSent', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        // jshint camelcase: false
        assert.equal(state.who.get(message2.sender.session_id).get('lastSent'), message2.time)
        assert.equal(state.who.get(message3.sender.session_id).get('lastSent'), message3.time)
        done()
      })
    })

    it('should update earliestLog', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        assert.equal(state.earliestLog, 'id1')
        done()
      })
    })
  }

  function testEditMessageEvent(type) {
    var deleteEvent = {
      'id': '0',
      'type': type,
      'data': _.merge({}, message1, {deleted: 12345}),
    }

    it('should update the message data in the tree', function(done) {
      chat.store.socketEvent({status: 'receive', body: logReply})
      handleSocket({status: 'receive', body: deleteEvent}, function(state) {
        assert(state.messages.get(message1.id).get('deleted') == 12345)
        done()
      })
    })
  }

  function testErrorLogging(type, error, done) {
    var errorEvent = {
      'type': type,
      'error': error,
    }
    handleSocket({status: 'receive', body: errorEvent}, function() {
      sinon.assert.calledOnce(console.warn)
      sinon.assert.calledWithExactly(console.warn, sinon.match.string, errorEvent.error)
      done()
    })
  }

  describe('received edit-message-event events', function() {
    testEditMessageEvent('edit-message-event')
  })

  describe('received edit-message-reply events', function() {
    testEditMessageEvent('edit-message-reply')

    it('should log a warning upon error', function(done) {
      testErrorLogging('edit-message-reply', 'oh no!', done)
    })
  })

  describe('received ban-reply events', function() {
    var banReplyEvent = {
      'id': '0',
      'type': 'ban-reply',
      'data': {
        'id': 'agent:tester2',
        'seconds': 60 * 60,
      },
    }

    it('should add the id to the banned ids set', function(done) {
      handleSocket({status: 'receive', body: banReplyEvent}, function(state) {
        assert(state.bannedIds.has(banReplyEvent.data.id))
        done()
      })
    })

    it('should log a warning upon error', function(done) {
      testErrorLogging('ban-reply', 'oops!', done)
    })
  })

  function checkMessagesChangedEvent(msgBody) {
    it('should trigger messagesChanged action', function(done) {
      chat.actions.messagesChanged.reset()
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        var ids = Immutable.Seq(msgBody.data.log).map(msg => msg.id).toArray()
        ids.unshift('__root')
        sinon.assert.calledOnce(chat.actions.messagesChanged)
        sinon.assert.calledWithExactly(chat.actions.messagesChanged, ids, state)
        done()
      })
    })
  }

  describe('received logs', function() {
    checkLogs(logReply)
    checkMessagesChangedEvent(logReply)

    it('should ignore empty logs', function(done) {
      var emptyLogReply = {
        'id': '0',
        'type': 'log-reply',
        'data': {
          'log': []
        }
      }

      handleSocket({status: 'receive', body: logReply}, function() {
        handleSocket({status: 'receive', body: emptyLogReply}, function(state) {
          assert.equal(state.messages.size, 3)
          done()
        })
      })
    })

    describe('receiving more logs', function() {
      it('messages should be added to logs', function(done) {
        handleSocket({status: 'receive', body: logReply}, function() {
          handleSocket({status: 'receive', body: moreLogReply}, function(state) {
            assert.equal(state.messages.size, logReply.data.log.length + 1)
            assert(state.messages.get('id0').isSuperset(Immutable.fromJS(message0)))
            done()
          })
        })
      })

      it('messages should all be assigned hues', function(done) {
        handleSocket({status: 'receive', body: logReply}, function() {
          handleSocket({status: 'receive', body: moreLogReply}, function(state) {
            assertMessagesHaveHues(state.messages)
            done()
          })
        })
      })

      it('messages should update sender lastSent', function(done) {
        handleSocket({status: 'receive', body: logReply}, function() {
          handleSocket({status: 'receive', body: moreLogReply}, function(state) {
            // jshint camelcase: false
            assert.equal(state.who.get(message0.sender.session_id).get('lastSent'), message0.time)
            done()
          })
        })
      })

      it('should update earliestLog', function(done) {
        handleSocket({status: 'receive', body: logReply}, function() {
          handleSocket({status: 'receive', body: moreLogReply}, function(state) {
            assert.equal(state.earliestLog, 'id0')
            done()
          })
        })
      })
    })

    describe('receiving redundant logs', function() {
      beforeEach(function() {
        chat.store.socketEvent({status: 'receive', body: logReply})
      })

      describe('should not change', function() {
        checkLogs(logReply)
      })

      it('should not trigger messagesChanged action', function(done) {
        var logReplyWithBefore = _.merge(_.clone(logReply), {data: {before: 'id0'}})
        chat.actions.messagesChanged.reset()
        handleSocket({status: 'receive', body: logReplyWithBefore}, function() {
          sinon.assert.notCalled(chat.actions.messagesChanged)
          done()
        })
      })
    })

    describe('loadMoreLogs action', function() {
      it('should not make a request if initial logs not loaded yet', function() {
        chat.store.loadMoreLogs()
        sinon.assert.notCalled(socket.send)
      })

      it('should request 50 more logs before the earliest message', function() {
        chat.store.socketEvent({status: 'receive', body: logReply})
        chat.store.loadMoreLogs()
        sinon.assert.calledWithExactly(socket.send, {
          type: 'log',
          data: {n: 50, before: 'id1'},
        })
      })

      it('should not make a request if one already in flight', function(done) {
        chat.store.socketEvent({status: 'receive', body: logReply})
        chat.store.loadMoreLogs()
        chat.store.loadMoreLogs()
        sinon.assert.calledOnce(socket.send)
        handleSocket({status: 'receive', body: moreLogReply}, function() {
          chat.store.loadMoreLogs()
          sinon.assert.calledTwice(socket.send)
          done()
        })
      })

      describe('status indicator', function() {
        beforeEach(function() {
          chat.store.socketEvent({status: 'receive', body: logReply})
        })

        it('should be set when loading more logs', function() {
          chat.store.loadMoreLogs()
          assert.equal(chat.store.state.loadingLogs, true)
        })

        it('should be reset 250ms after no more logs received', function() {
          chat.store.loadMoreLogs()
          assert.equal(chat.store.state.loadingLogs, true)
          clock.tick(1000)
          chat.store.socketEvent({status: 'receive', body: moreLogReply})
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
    it('users should be assigned to user list', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        assert.equal(state.who.size, whoReply.data.listing.length)
        assert(Immutable.Iterable(whoReply.data.listing).every(function(user) {
          // jshint camelcase: false
          var whoEntry = state.who.get(user.session_id)
          return !!whoEntry && whoEntry.isSuperset(Immutable.fromJS(user))
        }))
        done()
      })
    })

    it('users should all be assigned hues', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        assert(state.who.every(function(whoEntry) {
          return !!whoEntry.has('hue')
        }))
        done()
      })
    })
  }

  describe('received users', function() {
    checkUsers(whoReply)
  })

  describe('received snapshots', function() {
    checkLogs(snapshotReply)
    checkMessagesChangedEvent(snapshotReply)
    checkUsers(snapshotReply)

    it('should update server version', function(done) {
      handleSocket({status: 'receive', body: snapshotReply}, function(state) {
        assert.equal(state.serverVersion, snapshotReply.data.version)
        done()
      })
    })

    it('should update session id', function(done) {
      handleSocket({status: 'receive', body: snapshotReply}, function(state) {
        // jshint camelcase: false
        assert.equal(state.sessionId, snapshotReply.data.session_id)
        done()
      })
    })

    it('should set canJoin state to true', function(done) {
      handleSocket({status: 'receive', body: snapshotReply}, function(state) {
        assert.equal(state.canJoin, true)
        done()
      })
    })

    describe('on join', function() {
      beforeEach(function() {
        chat.store.joinRoom()
      })

      it('should set joined state to the join time', function(done) {
        handleSocket({status: 'receive', body: snapshotReply}, function(state) {
          assert.equal(state.joined, Date.now())
          done()
        })
      })

      it('should set auth type state to public if no bounce event received', function(done) {
        handleSocket({status: 'receive', body: snapshotReply}, function(state) {
          assert.equal(state.authType, 'public')
          done()
        })
      })

      it('should clear auth state', function(done) {
        chat.store.state.authState = 'trying-stored'
        handleSocket({status: 'receive', body: snapshotReply}, function(state) {
          assert.equal(state.authState, null)
          done()
        })
      })

      it('should trigger sending stored nick', function(done) {
        chat.store.state.roomName = 'ezzie'
        chat.store.storageChange(mockStorage)
        handleSocket({status: 'receive', body: snapshotReply}, function() {
          sinon.assert.calledWithExactly(socket.send, {
            type: 'nick',
            data: {name: mockStorage.room.ezzie.nick},
          })
          done()
        })
      })

      it('should not send stored nick if unset', function(done) {
        chat.store.state.roomName = 'ezzie'
        chat.store.storageChange({room: {}})
        handleSocket({status: 'receive', body: snapshotReply}, function() {
          sinon.assert.notCalled(socket.send)
          done()
        })
      })
    })
  })

  describe('received nick changes', function() {
    var rejectedNickReply = {
      'id': '1',
      'type': 'nick-reply',
      'error': 'error',
    }

    var nonexistentNickEvent = {
      'id': '2',
      'type': 'nick-event',
      'data': {
        'session_id': '32.64.96.128:54321',
        'id': 'agent:noman',
        'from': 'nonexistence',
        'to': 'absence',
      }
    }

    beforeEach(function() {
      chat.store.socketEvent({status: 'receive', body: helloEvent})
      chat.store.socketEvent({status: 'receive', body: snapshotReply})
    })

    it('should update user list name', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: nickReply}, function(state) {
          // jshint camelcase: false
          assert.equal(state.who.getIn([nickReply.data.session_id, 'name']), nickReply.data.to)
          done()
        })
      })
    })

    it('should update hue', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: nickReply}, function(state) {
          // jshint camelcase: false
          assert.equal(state.who.getIn([nickReply.data.session_id, 'hue']), 70)
          done()
        })
      })
    })

    it('should add nonexistent users', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: nonexistentNickEvent}, function(state) {
          // jshint camelcase: false
          assert(state.who.has(nonexistentNickEvent.data.session_id))
          done()
        })
      })
    })

    describe('in response to nick set', function() {
      it('should not update nick if rejected', function(done) {
        chat.store.state.nick = 'previous'
        chat.store.state.roomName = 'ezzie'
        handleSocket({status: 'receive', body: rejectedNickReply}, function(state) {
          assert.equal(state.nick, 'previous')
          done()
        })
      })

      it('should update stored nick', function(done) {
        chat.store.state.roomName = 'ezzie'
        handleSocket({status: 'receive', body: nickReply}, function(state) {
          assert.equal(state.nick, 'tester')
          sinon.assert.calledOnce(storage.setRoom)
          sinon.assert.calledWithExactly(storage.setRoom, 'ezzie', 'nick', 'tester')
          done()
        })
      })

      it('should update Raven user context', function(done) {
        handleSocket({status: 'receive', body: nickReply}, function() {
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

  describe('received join events', function() {
    var joinEvent = {
      'id': '1',
      'type': 'join-event',
      'data': {
        'session_id': '32.64.96.128:12347',
        'id': 'agent:someone',
        'name': '32.64.96.128:12347',
        'server_id': '1a2a3a4a5a6a',
        'server_era': '1b2b3b4b5b6b',
      }
    }

    it('should add to user list', function(done) {
      handleSocket({status: 'receive', body: joinEvent}, function(state) {
        // jshint camelcase: false
        assert(state.who.get(joinEvent.data.session_id).isSuperset(Immutable.fromJS(joinEvent.data)))
        done()
      })
    })

    it('should assign a hue', function(done) {
      handleSocket({status: 'receive', body: joinEvent}, function(state) {
        // jshint camelcase: false
        assert.equal(state.who.getIn([joinEvent.data.session_id, 'hue']), 50)
        done()
      })
    })
  })

  describe('received part events', function() {
    var partEvent = {
      'id': '1',
      'type': 'part-event',
      'data': {
        'session_id': '32.64.96.128:12345',
        'id': 'agent:tester1',
        'name': 'tester',
      },
    }

    it('should remove from user list', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: partEvent}, function(state) {
          // jshint camelcase: false
          assert(!state.who.has(partEvent.data.session_id))
          done()
        })
      })
    })
  })

  describe('setRoomSettings action', function() {
    it('should merge with roomSettings data', function() {
      chat.store.setRoomSettings({testing: true, another: {test: false}})
      assert.equal(chat.store.state.roomSettings.get('testing'), true)
      assert.equal(chat.store.state.roomSettings.getIn(['another', 'test']), false)
      chat.store.setRoomSettings({another: {test: true}})
      assert.equal(chat.store.state.roomSettings.getIn(['another', 'test']), true)
    })
  })

  describe('tryRoomPasscode action', function() {
    it('should set authData and send an auth attempt', function() {
      var testPassword = 'hunter2'
      chat.store.tryRoomPasscode(testPassword)
      assert.equal(chat.store.state.authData, testPassword)
      assert.equal(chat.store.state.authState, 'trying')
      sinon.assert.calledOnce(socket.send)
      sinon.assert.calledWithExactly(socket.send, {
        type: 'auth',
        data: {
          type: 'passcode',
          passcode: testPassword,
        },
      })
    })
  })

  describe('received bounce events', function() {
    it('should set passcode auth', function(done) {
      handleSocket({status: 'receive', body: bounceEvent}, function(state) {
        assert.equal(state.authType, 'passcode')
        done()
      })
    })

    it('should set canJoin state to false', function(done) {
      handleSocket({status: 'receive', body: bounceEvent}, function(state) {
        assert.equal(state.canJoin, false)
        done()
      })
    })

    describe('if not trying a stored passcode', function() {
      it('should set auth state to "needs-passcode"', function(done) {
        handleSocket({status: 'receive', body: bounceEvent}, function(state) {
          assert.equal(state.authState, 'needs-passcode')
          done()
        })
      })
    })

    describe('if trying a stored passcode', function() {
      it('should be ignored', function(done) {
        chat.store.state.authState = 'trying-stored'
        handleSocket({status: 'receive', body: bounceEvent}, function(state) {
          assert.equal(state.authState, 'trying-stored')
          done()
        })
      })
    })
  })

  describe('received auth reply events', function() {
    var incorrectAuthReplyEvent = {
      'id': '1',
      'type': 'auth-reply',
      'data': {
        'success': false,
        'reason': 'passcode incorrect',
      },
    }

    var errorAuthReplyEvent = {
      'id': '1',
      'type': 'auth-reply',
      'data': null,
      'error': 'command not implemented',
    }

    beforeEach(function() {
      chat.store.state.roomName = 'ezzie'
      chat.store.state.authType = 'passcode'
      chat.store.state.authData = 'hunter2'
    })

    describe('if successful', function() {
      it('should save auth data in storage', function(done) {
        handleSocket({status: 'receive', body: successfulAuthReplyEvent}, function(state) {
          sinon.assert.calledOnce(storage.setRoom)
          sinon.assert.calledWithExactly(storage.setRoom, 'ezzie', 'auth', {type: 'passcode', data: 'hunter2'})
          assert.equal(state.authState, null)
          done()
        })
      })
    })

    function testAuthFail(body) {
      describe('if stored auth unsuccessful', function() {
        it('should set auth state to "needs-passcode"', function() {
          chat.store.state.authState = 'trying-stored'
          handleSocket({status: 'receive', body: body}, function(state) {
            assert.equal(state.authState, 'needs-passcode')
          })
        })
      })

      describe('if auth unsuccessful', function() {
        it('should set auth state to "failed"', function() {
          chat.store.state.authState = 'trying'
          handleSocket({status: 'receive', body: body}, function(state) {
            assert.equal(state.authState, 'failed')
          })
        })
      })
    }

    describe('in case of error', function() {
      testAuthFail(errorAuthReplyEvent)
    })

    testAuthFail(incorrectAuthReplyEvent)
  })

  describe('received network partition events', function() {
    var networkPartitionEvent = {
      'id': '1',
      'type': 'network-event',
      'data': {
        'type': 'partition',
        'server_id': '1a2a3a4a5a6a',
        'server_era': '1b2b3b4b5b6b',
      },
    }

    it('should remove all associated users from the user list', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: networkPartitionEvent}, function(state) {
          assert.equal(state.who.size, 1)
          assert.equal(state.who.first().get('id'), whoReply.data.listing[2].id)
          done()
        })
      })
    })
  })

  describe('received ping events', function() {
    it('should be ignored', function() {
      var storeSpy = sinon.spy()
      support.listenOnce(chat.store, storeSpy)
      chat.store.socketEvent({status: 'receive', body: {type: 'ping-event'}})
      chat.store.socketEvent({status: 'receive', body: {type: 'ping-reply'}})
      sinon.assert.notCalled(storeSpy)
    })
  })

  describe('received unknown chat events', function() {
    var unknownEvent = {
      'id': '1',
      'type': 'wat-event',
      'data': {
        'wat': 'wat',
      },
    }

    it('should log a warning', function(done) {
      handleSocket({status: 'receive', body: unknownEvent}, function() {
        sinon.assert.calledOnce(console.warn)
        sinon.assert.calledWithExactly(console.warn, sinon.match.string, unknownEvent.type)
        done()
      })
    })
  })

  describe('received unknown socket events', function() {
    it('should log a warning', function() {
      chat.store.socketEvent({status: 'wat'})
      sinon.assert.calledOnce(console.warn)
      sinon.assert.calledWithExactly(console.warn, sinon.match.string, 'wat')
    })
  })
})
