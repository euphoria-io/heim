var support = require('./support/setup')
var _ = require('lodash')
var assert = require('assert')
var sinon = require('sinon')

describe('notification store', function() {
  var ChatTree = require('../lib/chat-tree')
  var notification = require('../lib/stores/notification')
  var storage = require('../lib/stores/storage')
  var ui = require('../lib/stores/ui')
  var clock
  var _Notification = window.Notification

  var startTime = 10 * 1000

  beforeEach(function() {
    clock = support.setupClock()
    clock.tick(startTime)
    sinon.stub(storage, 'set')
    sinon.stub(storage, 'setRoom')
    sinon.stub(Heim, 'setFavicon')
  })

  afterEach(function() {
    clock.restore()
    window.Notification = _Notification
    storage.set.restore()
    storage.setRoom.restore()
    Heim.setFavicon.restore()
  })

  describe('when unsupported', function() {
    beforeEach(function() {
      delete window.Notification
      support.resetStore(notification.store)
    })

    it('should set unsupported', function() {
      assert.equal(notification.store.getInitialState().popupsSupported, false)
    })
  })

  describe('when supported but not permitted', function() {
    beforeEach(function() {
      window.Notification = {popupsPermission: 'default'}
      support.resetStore(notification.store)
    })

    it('should set supported', function() {
      assert.equal(notification.store.getInitialState().popupsSupported, true)
    })

    it('should set no permission', function() {
      assert.equal(notification.store.getInitialState().popupsPermission, false)
    })

    describe('enabling popups', function() {
      beforeEach(function() {
        Notification.requestPermission = sinon.spy()
      })

      it('should request permission', function() {
        notification.store.enablePopups()
        sinon.assert.calledWithExactly(Notification.requestPermission, notification.store.onPermission)
      })
    })

    describe('receiving permission', function() {
      it('should set popupsPermission', function(done) {
        support.listenOnce(notification.store, function(state) {
          assert.equal(state.popupsPermission, true)
          done()
        })
        notification.store.onPermission('granted')
      })

      it('should store enabled', function() {
        notification.store.onPermission('granted')
        sinon.assert.calledWithExactly(storage.set, 'notify', true)
      })
    })

    describe('receiving denial', function() {
      it('should set no popupsPermission', function(done) {
        support.listenOnce(notification.store, function(state) {
          assert.equal(state.popupsPermission, false)
          done()
        })
        notification.store.onPermission('denied')
      })
    })
  })

  describe('when supported and permitted', function() {
    var message1 = {
      'id': 'id1',
      'time': 123456,
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'logan',
      },
      'content': 'hello, ezzie!',
    }

    var message2 = {
      'id': 'id2',
      'time': 123457,
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'ezzie',
      },
      'content': 'woof!',
    }

    var message2Own = _.merge({}, message2, {
      '_own': true,
    })

    var messageMention = {
      'id': 'id3',
      'time': 123457,
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'tester',
      },
      'content': 'hey @ezzie!',
      '_mention': true,
    }

    var messageMentionSeen = _.merge({}, messageMention, {
      '_seen': true,
    })

    var message2Reply1 = {
      'id': 'id4',
      'time': 123458,
      'parent': 'id2',
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'logan',
      },
      'content': 'kitty?',
    }

    var message2Reply2Own = {
      'id': 'id5',
      'time': 123459,
      'parent': 'id2',
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'ezzie',
      },
      'content': 'WOOF!',
      '_own': true,
    }

    var message2Reply3 = {
      'id': 'id6',
      'time': 123460,
      'parent': 'id2',
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'kitty',
      },
      'content': 'mew?',
    }

    var mockChatState = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
      ])
    }

    var mockChatState2 = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
      ])
    }

    var mockChatState2Own = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
        message2Own,
      ])
    }

    var mockChatStateMention = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        messageMention,
      ])
    }

    var mockChatStateMentionSeen = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        messageMentionSeen,
      ])
    }

    var mockChatState2Reply2Own = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        message2Reply1,
        message2Reply2Own,
      ])
    }

    var mockChatState2OwnReply = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
        message2Own,
        message2Reply1,
      ])
    }

    var mockChatState2Reply3 = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
        message2Own,
        message2Reply1,
        message2Reply2Own,
        message2Reply3,
      ])
    }

    var mockChatStateOrphan = {
      roomName: 'ezzie',
      messages: new ChatTree().reset([
        message1,
        message2Reply1,
      ])
    }

    var fakeNotification

    beforeEach(function() {
      window.Notification = sinon.spy(function() {
        this.close = sinon.spy(function() {
          if (this.onclose) {
            this.onclose()
          }
        })
        fakeNotification = this
      })
      Notification.permission = 'granted'
      support.resetStore(notification.store)
    })

    it('should set supported', function() {
      assert.equal(notification.store.getInitialState().popupsSupported, true)
    })

    it('should set popupsPermission', function() {
      assert.equal(notification.store.getInitialState().popupsPermission, true)
    })

    describe('enabling popups', function() {
      it('should store enabled and reset pause time', function() {
        notification.store.enablePopups()
        sinon.assert.calledWithExactly(storage.set, 'notify', true)
        sinon.assert.calledWithExactly(storage.set, 'notifyPausedUntil', null)
      })
    })

    describe('disabling popups', function() {
      it('should store disabled and reset pause time', function() {
        notification.store.disablePopups()
        sinon.assert.calledWithExactly(storage.set, 'notify', false)
        sinon.assert.calledWithExactly(storage.set, 'notifyPausedUntil', null)
      })
    })

    describe('disabling popups for a time', function() {
      it('should store pause time', function() {
        var time = startTime + 1000
        notification.store.pausePopupsUntil(time)
        sinon.assert.calledWithExactly(storage.set, 'notifyPausedUntil', time)
      })
    })

    describe('setting a room notification mode', function() {
      it('should store mode', function() {
        notification.store.setRoomNotificationMode('ezzie', 'none')
        sinon.assert.calledWithExactly(storage.setRoom, 'ezzie', 'notifyMode', 'none')
      })
    })

    describe('when disconnected', function() {
      it('should set favicon', function() {
        notification.store.chatStateChange({connected: false, joined: false})
        sinon.assert.calledOnce(Heim.setFavicon)
        sinon.assert.calledWithExactly(Heim.setFavicon, notification.favicons.disconnected)
      })
    })

    var storageMock = {notify: true, room: {ezzie: {notifyMode: 'message'}}}

    describe('when popups enabled', function() {
      beforeEach(function() {
        notification.store.chatStateChange({connected: true, joined: true})
        Heim.setFavicon.reset()
        notification.store.onInactive()
        notification.store.storageChange(storageMock)
      })

      describe('receiving a message before joined', function() {
        beforeEach(function() {
          notification.store.chatStateChange({connected: true, joined: false})
        })

        it('should add a notification and not display a popup', function(done) {
          notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
          support.listenOnce(notification.store, function() {
            sinon.assert.notCalled(Notification)
            assert.equal(notification.store.state.notifications.get('id1'), 'new-message')
            done()
          })
          clock.tick(0)
        })
      })

      describe('receiving a message while notifications paused', function() {
        it('should add a notification and not display a popup', function(done) {
          notification.store.storageChange(_.extend({}, storageMock, {notifyPausedUntil: startTime + 1000}))
          notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
          support.listenOnce(notification.store, function(state) {
            sinon.assert.notCalled(Notification)
            assert.equal(state.notifications.get('id1'), 'new-message')
            done()
          })
          clock.tick(0)
        })
      })

      function testNotifyModes(opts) {
        function test(expectPopup, mode) {
          describe('if notify mode is ' + JSON.stringify(mode) + '', function() {
            beforeEach(function() {
              notification.store.storageChange({notify: true, room: {ezzie: {notifyMode: mode}}})
              if (opts.messageReceivedId) {
                notification.store.messageReceived(opts.state.messages.get(opts.messageReceivedId), opts.state)
              }
              if (opts.messageChangedId) {
                notification.store.messagesChanged([opts.messageChangedId], opts.state)
              }
            })

            it('should set favicon', function(done) {
              support.listenOnce(notification.store, function() {
                sinon.assert.calledOnce(Heim.setFavicon)
                sinon.assert.calledWithExactly(Heim.setFavicon, opts.expectFavicon)
                done()
              })
              clock.tick(0)
            })

            it('should add notification', function(done) {
              support.listenOnce(notification.store, function(state) {
                assert.equal(state.notifications.get(opts.messageChangedId || opts.messageReceivedId), opts.expectKind)
                done()
              })
              clock.tick(0)
            })

            it('should ' + (expectPopup ? '' : 'not ') + 'display a popup', function(done) {
              support.listenOnce(notification.store, function() {
                if (expectPopup) {
                  sinon.assert.calledOnce(Notification)
                  sinon.assert.calledWithExactly(Notification, 'ezzie', {
                    icon: opts.expectIcon,
                    body: opts.expectBody,
                  })
                } else {
                  sinon.assert.notCalled(Notification)
                }
                done()
              })
              clock.tick(0)
            })
          })
        }

        _.each(opts.modes, test)
        test(opts.modes.mention, undefined)
      }

      describe('receiving a message', function() {
        testNotifyModes({
          modes: {
            message: true,
            reply: false,
            mention: false,
            none: false,
          },
          messageReceivedId: 'id1',
          state: mockChatState,
          expectFavicon: notification.favicons.active,
          expectIcon: notification.icons.active,
          expectBody: 'logan: hello, ezzie!',
          expectKind: 'new-message',
        })
      })

      describe('receiving the same message again', function() {
        it('should not display a popup', function(done) {
          notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
          clock.tick(0)
          fakeNotification.close()
          notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
          support.listenOnce(notification.store, function(state) {
            sinon.assert.calledOnce(Notification)
            assert.equal(state.notifications.get(message1.id), 'new-message')
            done()
          })
          clock.tick(0)
        })
      })

      describe('closing and receiving a new message', function() {
        it('should display a second popup', function(done) {
          notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
          clock.tick(0)
          fakeNotification.close()
          notification.store.messageReceived(mockChatState2.messages.get('id2'), mockChatState2)
          support.listenOnce(notification.store, function(state) {
            sinon.assert.calledTwice(Notification)
            assert.equal(state.notifications.get(message1.id), 'new-message')
            done()
          })
          clock.tick(0)
        })
      })

      describe('updating a child reply', function() {
        describe('when joined', function() {
          testNotifyModes({
            modes: {
              message: true,
              reply: true,
              mention: false,
              none: false,
            },
            messageChangedId: message2Reply1.id,
            state: mockChatState2OwnReply,
            expectFavicon: notification.favicons.active,
            expectIcon: notification.icons.active,
            expectBody: 'logan: kitty?',
            expectKind: 'new-reply',
          })

          it('should replace an existing new-message notification', function() {
            notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
            clock.tick(0)
            assert.equal(notification.store.alerts['new-message'].messageId, message1.id)
            sinon.assert.calledOnce(Notification)
            var messageNotification = fakeNotification

            notification.store.messagesChanged([message2Reply1.id], mockChatState2OwnReply)
            clock.tick(0)
            assert.equal(notification.store.alerts['new-message'].messageId, message2Reply1.id)
            sinon.assert.calledOnce(messageNotification.close)
            sinon.assert.calledTwice(Notification)
          })
        })
      })

      describe('updating a sibling reply', function() {
        describe('when joined', function() {
          testNotifyModes({
            modes: {
              message: true,
              reply: true,
              mention: false,
              none: false,
            },
            messageChangedId: message2Reply3.id,
            state: mockChatState2Reply3,
            expectFavicon: notification.favicons.active,
            expectIcon: notification.icons.active,
            expectBody: 'kitty: mew?',
            expectKind: 'new-reply',
          })
        })
      })

      describe('updating a mention', function() {
        describe('when joined', function() {
          testNotifyModes({
            modes: {
              message: true,
              reply: true,
              mention: true,
              none: false,
            },
            messageReceivedId: messageMention.id,
            messageChangedId: messageMention.id,
            state: mockChatStateMention,
            expectFavicon: notification.favicons.highlight,
            expectIcon: notification.icons.highlight,
            expectBody: 'tester: hey @ezzie!',
            expectKind: 'new-mention',
          })

          it('if seen before should not add a notification and not display a popup', function() {
            notification.store.messagesChanged([messageMention.id], mockChatStateMentionSeen)
            clock.tick(0)
            sinon.assert.notCalled(Notification)
            sinon.assert.notCalled(Heim.setFavicon)
            assert(!notification.store.state.notifications.has('id3'))
          })

          describe('and dismissing it and updating the mention again', function() {
            it('should not create another notification', function(done) {
              notification.store.messagesChanged([messageMention.id], mockChatStateMention)
              support.listenOnce(notification.store, function(state) {
                assert.equal(state.notifications.get(messageMention.id), 'new-mention')
                notification.store.dismissNotification(messageMention.id)
                assert(!state.notifications.has(messageMention.id))
                notification.store.messagesChanged([messageMention.id], mockChatStateMention)
                clock.tick(0)
                assert(!notification.store.state.notifications.has(messageMention.id))
                done()
              })
              clock.tick(0)
            })
          })
        })

        describe('receiving multiple messages', function() {
          it('should display a popup for the latest message', function(done) {
            notification.store.messageReceived(mockChatState2Reply3.messages.get(message2Reply1.id), mockChatState2Reply3)
            notification.store.messageReceived(mockChatState2Reply3.messages.get(message2Reply3.id), mockChatState2Reply3)
            support.listenOnce(notification.store, function() {
              sinon.assert.calledOnce(Notification)
              sinon.assert.calledWithExactly(Notification, 'ezzie', {
                icon: notification.icons.active,
                body: 'kitty: mew?',
              })
              done()
            })
            clock.tick(0)
          })
        })

        describe('when not joined', function() {
          it('should not display a popup', function(done) {
            notification.store.chatStateChange({connected: true, joined: false})
            notification.store.messagesChanged([messageMention.id], mockChatStateMention)
            support.listenOnce(notification.store, function(state) {
              sinon.assert.notCalled(Notification)
              sinon.assert.calledWithExactly(Heim.setFavicon, '/static/favicon.png')
              assert.equal(state.notifications.get(messageMention.id), 'new-mention')
              done()
            })
            clock.tick(0)
          })
        })
      })

      describe('updating an orphaned message', function() {
        describe('when joined', function() {
          it('should not set favicon, add a notification, or display a popup', function() {
            notification.store.messagesChanged([message2Reply1.id], mockChatStateOrphan)
            clock.tick(0)
            sinon.assert.notCalled(Heim.setFavicon)
            assert(!notification.store.state.notifications.has(messageMention.id))
            sinon.assert.notCalled(Notification)
          })
        })
      })

      describe('removing an alert with mismatched message id', function() {
        it('should have no effect', function() {
          notification.store.messagesChanged([messageMention.id], mockChatStateMention)
          clock.tick(0)
          sinon.assert.calledOnce(Notification)
          notification.store.removeAlert('new-mention', 'nonexistent')
          sinon.assert.notCalled(fakeNotification.close)
        })
      })

      // workaround for Chrome behavior
      describe('closing an alert when the browser that doesn\'t call onclose consistently', function() {
        it('should try again in 500ms', function() {
          notification.store.messagesChanged([messageMention.id], mockChatStateMention)
          clock.tick(0)
          sinon.assert.calledOnce(Notification)
          fakeNotification.close = sinon.spy()
          notification.store.closePopup('new-mention')
          sinon.assert.calledOnce(fakeNotification.close)
          fakeNotification.close.reset()
          clock.tick(500)
          sinon.assert.calledOnce(fakeNotification.close)
        })

        it('should not try again in 500ms if onclose was called', function() {
          notification.store.messagesChanged([messageMention.id], mockChatStateMention)
          clock.tick(0)
          sinon.assert.calledOnce(Notification)
          notification.store.closePopup('new-mention')
          sinon.assert.calledOnce(fakeNotification.close)
          fakeNotification.close.reset()
          clock.tick(500)
          sinon.assert.notCalled(fakeNotification.close)
        })
      })
    })

    describe('own messages', function() {
      beforeEach(function() {
        notification.store.chatStateChange({connected: true, joined: true})
        sinon.stub(notification, 'dismissNotification')
      })

      afterEach(function() {
        notification.dismissNotification.restore()
      })

      it('should dismiss notifications on parent and sibling messages', function() {
        notification.store.messageReceived(mockChatState2Reply2Own.messages.get('id5'), mockChatState2Reply2Own)
        sinon.assert.calledWithExactly(notification.dismissNotification, message2.id)
        sinon.assert.calledWithExactly(notification.dismissNotification, message2Reply1.id)
      })

      it('should not dismiss notifications on top level messages (children of root)', function() {
        notification.store.messageReceived(mockChatState2Own.messages.get('id2'), mockChatState2Own)
        sinon.assert.notCalled(notification.dismissNotification)
      })
    })

    describe('activity tracking', function() {
      beforeEach(function() {
        notification.store.chatStateChange({connected: true, joined: true})
        Heim.setFavicon.reset()
      })

      it('should start active', function() {
        assert.equal(notification.store.active, true)
      })

      it('should set active when page becomes active', function() {
        notification.store.onActive()
        assert.equal(notification.store.active, true)
      })

      it('should set inactive when page becomes inactive', function() {
        notification.store.onInactive()
        assert.equal(notification.store.active, false)
      })

      it('should not open popups when active', function() {
        notification.store.onActive()
        notification.store.storageChange(storageMock)
        notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
        clock.tick(0)
        sinon.assert.notCalled(Notification)
      })

      it('should close popup when window becomes active', function() {
        notification.store.onInactive()
        notification.store.storageChange(storageMock)
        notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
        clock.tick(0)
        sinon.assert.calledOnce(Notification)
        notification.store.onActive()
        sinon.assert.calledOnce(fakeNotification.close)
      })

      it('should reset favicon when window becomes active', function() {
        notification.store.onInactive()
        notification.store.storageChange(storageMock)
        notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
        clock.tick(0)
        sinon.assert.calledOnce(Heim.setFavicon)
        Heim.setFavicon.reset()

        notification.store.onActive()
        sinon.assert.calledOnce(Heim.setFavicon)
        sinon.assert.calledWithExactly(Heim.setFavicon, '/static/favicon.png')
      })
    })

    describe('with a popup showing', function() {
      beforeEach(function() {
        notification.store.chatStateChange({connected: true, joined: true})
        Heim.setFavicon.reset()
        notification.store.onInactive()
        notification.store.storageChange(storageMock)
        notification.store.messageReceived(mockChatState.messages.get('id1'), mockChatState)
        clock.tick(0)
        sinon.stub(ui, 'gotoMessageInPane')
        window.uiwindow = {focus: sinon.stub()}
      })

      afterEach(function() {
        ui.gotoMessageInPane.restore()
        delete window.uiwindow
      })

      it('should replace with another popup', function() {
        notification.store.messageReceived(mockChatState2.messages.get('id2'), mockChatState2)
        clock.tick(0)
        sinon.assert.calledTwice(Notification)
      })

      it('should focus window and go to message when clicked', function() {
        fakeNotification.onclick()
        sinon.assert.calledOnce(window.uiwindow.focus)
        sinon.assert.calledOnce(ui.gotoMessageInPane)
        sinon.assert.calledWithExactly(ui.gotoMessageInPane, 'id1')
      })

      it('should close after 3 seconds', function() {
        clock.tick(3000)
        sinon.assert.calledOnce(fakeNotification.close)
      })

      it('should retain favicon state after timeout and reconnect', function() {
        clock.tick(3000)
        notification.store.chatStateChange({connected: false, joined: false})
        Heim.setFavicon.reset()
        notification.store.chatStateChange({connected: true, joined: true})
        sinon.assert.calledOnce(Heim.setFavicon)
        sinon.assert.calledWithExactly(Heim.setFavicon, notification.favicons.active)
      })

      describe('when window unloading', function() {
        it('should close the popup', function() {
          notification.store.removeAllAlerts()
          sinon.assert.calledOnce(fakeNotification.close)
        })
      })
    })

    describe('dismissing a nonexistent notification', function() {
      it('should have no effect', function() {
        assert(!notification.store.state.notifications.has('nonexistent'))
        notification.store.dismissNotification('nonexistent')
      })
    })

    describe('closing a nonexistent alert', function() {
      it('should have no effect', function() {
        assert(!_.has(notification.store.alerts, 'new-mention'))
        notification.store.closePopup('new-mention')
      })
    })
  })
})
