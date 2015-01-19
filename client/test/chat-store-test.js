var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')
var Immutable = require('immutable')


describe('chat store', function() {
  var actions = require('../lib/actions')
  var chat = require('../lib/stores/chat')
  var socket = require('../lib/stores/socket')
  var storage = require('../lib/stores/storage')

  beforeEach(function() {
    sinon.stub(socket, 'send')
    support.resetStore(chat.store)
  })

  afterEach(function() {
    socket.send.restore()
  })

  function handleSocket(ev, callback) {
    support.listenOnce(chat.store, callback)
    chat.store.socketEvent(ev)
  }

  function checkWhoSorted(who) {
    var sorted = who.sortBy(function(user) { return user.get('name') })
    assert(who.equals(sorted))
  }

  var message1 = {
    'id': 'id1',
    'time': 123456,
    'sender': {
      'id': '32.64.96.128:12345',
      'name': 'tester',
    },
    'content': 'test',
  }

  var message2 = {
    'id': 'id2',
    'time': 123457,
    'sender': {
      'id': '32.64.96.128:12345',
      'name': 'tester',
    },
    'content': 'test2',
  }

  var message3 = {
    'id': 'id3',
    'parent': 'id2',
    'time': 123458,
    'sender': {
      'id': '32.64.96.128:12346',
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
    'time': 123456,
    'sender': {
      'id': '32.64.96.128:12345',
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
          'id': '32.64.96.128:12344',
          'name': '000tester',
        },
        {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        {
          'id': '32.64.96.128:12346',
          'name': 'tester2',
        },
      ]
    }
  }

  var snapshotReply = {
    'id': '',
    'type': 'snapshot-event',
    'data': {
      'listing': whoReply.data.listing,
      'log': logReply.data.log,
    }
  }

  it('should initialize with null connected state', function() {
    assert.equal(chat.store.getInitialState().connected, null)
  })

  it('should initialize with empty collections', function() {
    var initialState = chat.store.getInitialState()
    assert.equal(initialState.messages.size, 0)
    assert.equal(initialState.who.size, 0)
    assert.deepEqual(initialState.nickHues, {})
  })

  describe('connect action', function() {
    beforeEach(function() {
      sinon.stub(socket, 'connect')
    })

    afterEach(function() {
      socket.connect.restore()
    })

    it('should connect socket', function() {
      chat.store.connect()
      assert(socket.connect.called)
    })
  })

  describe('setNick action', function() {
    var testNick = 'test-nick'

    beforeEach(function() {
      sinon.stub(storage, 'set')
      chat.store.setNick(testNick)
    })

    afterEach(function() {
      storage.set.restore()
    })

    it('should send a nick change', function() {
      sinon.assert.calledWithExactly(socket.send, {
        type: 'nick',
        data: {name: testNick},
      })
    })

    it('should update stored nick', function() {
      sinon.assert.calledWithExactly(storage.set, 'nick', testNick)
    })

    it('should avoid re-sending same nick', function() {
      chat.store.storageChange({nick: testNick})
      chat.store.setNick(testNick)
      assert(socket.send.calledOnce)
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
      var testContent = 'hello, ezzie!'
      chat.store.sendMessage(testContent, '123test')
      sinon.assert.calledWithExactly(socket.send, {
        type: 'send',
        data: {content: testContent, parent: '123test'},
      })
    })
  })

  describe('setEntryText action', function() {
    it('should update entryText', function(done) {
      var text = 'hello, ezzie!'

      support.listenOnce(chat.store, function(state) {
        assert.equal(state.entryText, text)
        done()
      })

      chat.store.setEntryText(text)
    })
  })

  describe('toggleFocusMessage action', function() {
    beforeEach(function() {
      sinon.stub(actions, 'focusMessage')
    })

    afterEach(function() {
      actions.focusMessage.restore()
    })

    describe('on a top-level message', function() {
      describe('if not already focused', function() {
        it('should focus', function() {
          chat.store.toggleFocusMessage('id1', '__root')
          sinon.assert.calledOnce(actions.focusMessage)
          sinon.assert.calledWithExactly(actions.focusMessage, 'id1')
        })
      })

      describe('if already focused', function() {
        it('should reset focus', function() {
          chat.store.state.focusedMessage = 'id1'
          chat.store.toggleFocusMessage('id1', '__root')
          sinon.assert.calledOnce(actions.focusMessage)
          sinon.assert.calledWithExactly(actions.focusMessage, null)
        })
      })
    })

    describe('on a child message', function() {
      describe('if parent not already focused', function() {
        it('should focus parent', function() {
          chat.store.toggleFocusMessage('id2', 'id1')
          sinon.assert.calledOnce(actions.focusMessage)
          sinon.assert.calledWithExactly(actions.focusMessage, 'id1')
        })

        it('should focus child', function() {
          chat.store.state.focusedMessage = 'id1'
          chat.store.toggleFocusMessage('id2', 'id1')
          sinon.assert.calledOnce(actions.focusMessage)
          sinon.assert.calledWithExactly(actions.focusMessage, 'id2')
        })
      })
    })
  })

  describe('when connected', function() {
    it('should have connected state: true', function() {
      handleSocket({status: 'open'}, function(state) {
        assert.equal(state.connected, true)
      })
    })

    it('should send stored nick upon connecting', function(done) {
      var mockStorage = {
        nick: 'test-nick',
      }
      chat.store.storageChange(mockStorage)
      handleSocket({status: 'open'}, function() {
        sinon.assert.calledWithExactly(socket.send, {
          type: 'nick',
          data: {name: mockStorage.nick},
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
  })

  describe('received messages', function() {
    var sendEvent = {
      'id': '0',
      'type': 'send-event',
      'data': {
        'id': 'id1',
        'time': 123456,
        'sender': {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        'content': 'test',
      }
    }

    var sendReplyEvent = {
      'id': '1',
      'type': 'send-event',
      'data': {
        'id': 'id2',
        'parent': 'id1',
        'time': 123456,
        'sender': {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        'content': 'test',
      }
    }

    it('should be appended to log', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function(state) {
        assert(state.messages.last().isSuperset(Immutable.fromJS(sendEvent.data)))
        done()
      })
    })

    it('should be assigned a hue', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function(state) {
        assert.equal(state.messages.last().getIn(['sender', 'hue']), 153)
        done()
      })
    })

    it('should be stored as children of parent', function(done) {
      handleSocket({status: 'receive', body: sendEvent}, function() {
        handleSocket({status: 'receive', body: sendReplyEvent}, function(state) {
          assert(state.messages.get('id1').get('children').contains('id2'))
          done()
        })
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

    it('should update earliestLog', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        assert.equal(state.earliestLog, 'id1')
        done()
      })
    })
  }

  describe('received logs', function() {
    checkLogs(logReply)

    it('should ignore empty logs', function(done) {
      var emptyLogReply = {
        'id': '0',
        'type': 'log-reply',
        'data': {
          'log': []
        }
      }

      handleSocket({status: 'receive', body: emptyLogReply}, function(state) {
        assert.equal(state.messages.size, 0)
        done()
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

      it('should persist focusedMessage state', function(done) {
        chat.store.state.nick = 'test'
        support.listenOnce(chat.store, function(state) {
          assert.equal(state.messages.get('id1').get('entry'), true)

          support.listenOnce(chat.store, function(state) {
            assert.equal(state.messages.get('id1').get('entry'), true)
            done()
          })

          chat.store.socketEvent({status: 'receive', body: logReply})
        })

        chat.store.focusMessage('id1')
      })
    })

    describe('focusMessage action', function() {
      beforeEach(function() {
        chat.store.state.nick = 'test'
        chat.store.socketEvent({status: 'receive', body: logReply})
      })

      it('should enable entry on specified message and disable entry on previously focused message', function(done) {
        support.listenOnce(chat.store, function(state) {
          assert.equal(state.messages.get('id1').get('entry'), true)

          support.listenOnce(chat.store, function(state) {
            assert.equal(state.messages.get('id1').get('entry'), false)
            assert.equal(state.messages.get('id2').get('entry'), true)
            done()
          })

          chat.store.focusMessage('id2')
        })

        chat.store.focusMessage('id1')
      })

      it('should update focusedMessage value', function(done) {
        support.listenOnce(chat.store, function(state) {
          assert.equal(state.focusedMessage, 'id1')
          done()
        })

        chat.store.focusMessage('id1')
      })

      it('should not update if specified message already focused', function() {
        sinon.stub(chat.store, 'trigger')
        chat.store.focusMessage('id1')
        chat.store.focusMessage('id1')
        sinon.assert.calledOnce(chat.store.trigger)
        chat.store.trigger.restore()
      })

      it('should not update if no nick set', function() {
        chat.store.state.nick = null
        sinon.stub(chat.store, 'trigger')
        chat.store.focusMessage('id1')
        sinon.assert.notCalled(chat.store.trigger)
        chat.store.trigger.restore()
      })
    })

    describe('loadMoreLogs action', function() {
      it('should not make a request if initial logs not loaded yet', function() {
        chat.store.loadMoreLogs()
        sinon.assert.notCalled(socket.send)
      })

      it('should request 100 more logs before the earliest message', function() {
        chat.store.socketEvent({status: 'receive', body: logReply})
        chat.store.loadMoreLogs()
        sinon.assert.calledWithExactly(socket.send, {
          type: 'log',
          data: {n: 100, before: 'id1'},
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
    })
  })

  function checkUsers(msgBody) {
    it('users should be assigned to user list', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        assert.equal(state.who.size, whoReply.data.listing.length)
        assert(Immutable.Iterable(whoReply.data.listing).every(function(user) {
          var whoEntry = state.who.get(user.id)
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

    it('users should be sorted by name', function(done) {
      handleSocket({status: 'receive', body: msgBody}, function(state) {
        checkWhoSorted(state.who)
        done()
      })
    })
  }

  describe('received users', function() {
    checkUsers(whoReply)
  })

  describe('received snapshots', function() {
    checkLogs(snapshotReply)
    checkUsers(snapshotReply)
  })

  describe('received nick changes', function() {
    var nickReply = {
      'id': '1',
      'type': 'nick-reply',
      'data': {
        'id': '32.64.96.128:12345',
        'from': 'tester',
        'to': 'tester3',
      }
    }

    var nonexistentNickEvent = {
      'id': '2',
      'type': 'nick-event',
      'data': {
        'id': '32.64.96.128:54321',
        'from': 'nonexistence',
        'to': 'absence',
      }
    }

    it('should update user list name', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: nickReply}, function(state) {
          assert.equal(state.who.getIn([nickReply.data.id, 'name']), nickReply.data.to)
          done()
        })
      })
    })

    it('should update hue', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: nickReply}, function(state) {
          assert.equal(state.who.getIn([nickReply.data.id, 'hue']), 204)
          done()
        })
      })
    })

    it('should keep the list sorted', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: nickReply}, function(state) {
          checkWhoSorted(state.who)
          done()
        })
      })
    })

    it('should add nonexistent users', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: nonexistentNickEvent}, function(state) {
          assert(state.who.has(nonexistentNickEvent.data.id))
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
        'id': '32.64.96.128:12347',
        'name': '32.64.96.128:12347',
      }
    }

    it('should add to user list', function(done) {
      handleSocket({status: 'receive', body: joinEvent}, function(state) {
        assert(state.who.get(joinEvent.data.id).isSuperset(Immutable.fromJS(joinEvent.data)))
        done()
      })
    })

    it('should assign a hue', function(done) {
      handleSocket({status: 'receive', body: joinEvent}, function(state) {
        assert.equal(state.who.getIn([joinEvent.data.id, 'hue']), 161)
        done()
      })
    })

    it('should keep the list sorted', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: joinEvent}, function(state) {
          checkWhoSorted(state.who)
          done()
        })
      })
    })
  })

  describe('received part events', function() {
    var partEvent = {
      'id': '1',
      'type': 'part-event',
      'data': {
        'id': '32.64.96.128:12345',
        'name': 'tester',
      },
    }

    it('should remove from user list', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: partEvent}, function(state) {
          assert(!state.who.has(partEvent.data.id))
          done()
        })
      })
    })
  })
})
