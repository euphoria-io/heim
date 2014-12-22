var support = require('./support/setup')
var assert = require('assert')
var sinon = require('sinon')
var Immutable = require('immutable')


describe('chat store', function() {
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
    it('should send a nick change', function() {
      var testContent = 'hello, ezzie!'
      chat.store.sendMessage(testContent)
      sinon.assert.calledWithExactly(socket.send, {
        type: 'send',
        data: {content: testContent},
      })
    })
  })

  describe('when connected', function() {
    it('should have connected state: true', function() {
      handleSocket({status: 'open'}, function(state) {
        assert.equal(state.connected, true)
      })
    })

    it('should fetch logs and users upon connecting', function(done) {
      handleSocket({status: 'open'}, function() {
        sinon.assert.calledWithExactly(socket.send, {
          type: 'log',
          data: {n: 1000},
        })
        sinon.assert.calledWithExactly(socket.send, {
          type: 'who',
        })
        done()
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
    var sendReply = {
      'id': '0',
      'type': 'send-reply',
      'data': {
        'time': 123456,
        'sender': {
          'id': '32.64.96.128:12345',
          'name': 'tester',
        },
        'content': 'test',
      }
    }

    it('should be appended to log', function(done) {
      handleSocket({status: 'receive', body: sendReply}, function(state) {
        assert(state.messages.last().isSuperset(Immutable.fromJS(sendReply.data)))
        done()
      })
    })

    it('should be assigned a hue', function(done) {
      handleSocket({status: 'receive', body: sendReply}, function(state) {
        assert.equal(state.messages.last().getIn(['sender', 'hue']), 153)
        done()
      })
    })
  })

  describe('received logs', function() {
    var logReply = {
      'id': '0',
      'type': 'log-reply',
      'data': {
        'log': [
          {
            'time': 123456,
            'sender': {
              'id': '32.64.96.128:12345',
              'name': 'tester',
            },
            'content': 'test',
          },
          {
            'time': 123457,
            'sender': {
              'id': '32.64.96.128:12345',
              'name': 'tester',
            },
            'content': 'test2',
          },
          {
            'time': 123458,
            'sender': {
              'id': '32.64.96.128:12346',
              'name': 'tester2',
            },
            'content': 'test3',
          },
        ]
      }
    }

    it('should be assigned to log', function(done) {
      handleSocket({status: 'receive', body: logReply}, function(state) {
        assert.equal(state.messages.size, logReply.data.log.length)
        assert(state.messages.every(function(message, idx) {
          return message.isSuperset(Immutable.fromJS(logReply.data.log[idx]))
        }))
        done()
      })
    })

    it('should all be assigned hues', function(done) {
      handleSocket({status: 'receive', body: logReply}, function(state) {
        assert(state.messages.every(function(message) {
          return message.hasIn(['sender', 'hue'])
        }))
        done()
      })
    })
  })

  describe('received users', function() {
    var whoReply = {
      'id': '0',
      'type': 'who-reply',
      'data': {
        'listing': [
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

    it('should be assigned to user list', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function(state) {
        assert.equal(state.who.size, whoReply.data.listing.length)
        assert(Immutable.Iterable(whoReply.data.listing).every(function(user) {
          var whoEntry = state.who.get(user.id)
          return !!whoEntry && whoEntry.isSuperset(Immutable.fromJS(user))
        }))
        done()
      })
    })

    it('should all be assigned hues', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function(state) {
        assert(state.who.every(function(whoEntry) {
          return !!whoEntry.has('hue')
        }))
        done()
      })
    })

    it('should be sorted by name', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function(state) {
        var sorted = state.who.sortBy(function(user) { return user.get('name') })
        assert(state.who.equals(sorted))
        done()
      })
    })
  })

  describe('received nick changes', function() {
    var whoReply = {
      'id': '0',
      'type': 'who-reply',
      'data': {
        'listing': [
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

    it('should re-sort the list', function(done) {
      handleSocket({status: 'receive', body: whoReply}, function() {
        handleSocket({status: 'receive', body: nickReply}, function(state) {
          var sorted = state.who.sortBy(function(user) { return user.get('name') })
          assert(state.who.equals(sorted))
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
})
