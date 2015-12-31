import support from './support/setup'
import _ from 'lodash'
import assert from 'assert'
import sinon from 'sinon'

import ChatTree from '../lib/ChatTree'
import notification from '../lib/stores/notification'
import storage from '../lib/stores/storage'
import ui from '../lib/stores/ui'


describe('notification store', () => {
  const _Notification = window.Notification
  let startTime
  let clock

  beforeEach(() => {
    clock = support.setupClock()
    clock.tick()
    startTime = Date.now()
    sinon.stub(storage, 'set')
    sinon.stub(storage, 'setRoom')
    sinon.stub(Heim, 'setFavicon')
    sinon.stub(Heim, 'setTitleMsg')
  })

  afterEach(() => {
    clock.restore()
    window.Notification = _Notification
    storage.set.restore()
    storage.setRoom.restore()
    Heim.setFavicon.restore()
    Heim.setTitleMsg.restore()
  })

  describe('when unsupported', () => {
    beforeEach(() => {
      delete window.Notification
      support.resetStore(notification.store)
    })

    it('should set unsupported', () => {
      assert.equal(notification.store.getInitialState().popupsSupported, false)
    })
  })

  describe('when supported but not permitted', () => {
    beforeEach(() => {
      window.Notification = {popupsPermission: 'default'}
      support.resetStore(notification.store)
    })

    it('should set supported', () => {
      assert.equal(notification.store.getInitialState().popupsSupported, true)
    })

    it('should set no permission', () => {
      assert.equal(notification.store.getInitialState().popupsPermission, false)
    })

    describe('enabling popups', () => {
      beforeEach(() => {
        Notification.requestPermission = sinon.spy()
      })

      it('should request permission', () => {
        notification.store.enablePopups()
        sinon.assert.calledWithExactly(Notification.requestPermission, notification.store.onPermission)
      })
    })

    describe('receiving permission', () => {
      it('should set popupsPermission', done => {
        support.listenOnce(notification.store, state => {
          assert.equal(state.popupsPermission, true)
          done()
        })
        notification.store.onPermission('granted')
      })

      it('should store enabled', () => {
        notification.store.onPermission('granted')
        sinon.assert.calledWithExactly(storage.set, 'notify', true)
      })
    })

    describe('receiving denial', () => {
      it('should set no popupsPermission', done => {
        support.listenOnce(notification.store, state => {
          assert.equal(state.popupsPermission, false)
          done()
        })
        notification.store.onPermission('denied')
      })
    })
  })

  describe('when supported and permitted', () => {
    const message1 = {
      'id': 'id1',
      'time': 123456,
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'logan',
      },
      'content': 'hello, ezzie!',
    }

    const message2 = {
      'id': 'id2',
      'time': 123457,
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'ezzie',
      },
      'content': 'woof!',
    }

    const message3 = {
      'id': 'id3',
      'time': 123458,
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'max',
      },
      'content': 'whoa',
    }

    const messageOld = {
      'id': 'id0',
      'time': 123450,
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'ezzie',
      },
      'content': '/me yawns',
    }

    const message2Own = _.merge({}, message2, {
      '_own': true,
    })

    const messageMention = {
      'id': 'id3',
      'time': 123457,
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'tester',
      },
      'content': 'hey @ezzie!',
      '_mention': true,
    }

    const messageMentionSeen = _.merge({}, messageMention, {
      '_seen': true,
    })

    const messageMentionShadow = {
      'id': 'id3',
      'parent': null,
      '_mention': true,
    }

    const messageMentionDeleted = _.merge({}, messageMention, {
      'deleted': true,
    })

    const message2Reply1 = {
      'id': 'id4',
      'time': 123458,
      'parent': 'id2',
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'logan',
      },
      'content': 'kitty?',
    }

    const message2Reply2Own = {
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

    const message2Reply3 = {
      'id': 'id6',
      'time': 123460,
      'parent': 'id2',
      'sender': {
        'id': '32.64.96.128:12345',
        'name': 'kitty',
      },
      'content': 'mew?',
    }

    const emptyChatState = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree(),
    }

    const mockChatState = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
      ]),
    }

    const mockChatState2 = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
      ]),
    }

    const mockChatState3 = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        message3,
      ]),
    }

    const mockChatState3Old = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        messageOld,
        message1,
        message2,
        message3,
      ]),
    }

    const mockChatState2Own = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2Own,
      ]),
    }

    const mockChatStateMention = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        messageMention,
      ]),
    }

    const mockChatStateMentionSeen = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        messageMentionSeen,
      ]),
    }

    const mockChatStateMentionShadow = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        messageMentionShadow,
      ]),
    }

    const mockChatStateMentionDeleted = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        messageMentionDeleted,
      ]),
    }

    const mockChatState2Reply2Own = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2,
        message2Reply1,
        message2Reply2Own,
      ]),
    }

    const mockChatState2OwnReply = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2Own,
        message2Reply1,
      ]),
    }

    const mockChatState2Reply3 = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2Own,
        message2Reply1,
        message2Reply2Own,
        message2Reply3,
      ]),
    }

    const mockChatStateOrphan = {
      roomName: 'ezzie',
      roomTitle: '&ezzie',
      messages: new ChatTree().reset([
        message1,
        message2Reply1,
      ]),
    }

    const storageMock = {
      notify: true,
      notifyPausedUntil: 0,
      room: {ezzie: {notifyMode: 'message'}},
    }

    function simulateMessages(ids, state) {
      notification.store.messagesChanged(ids, state)
      _.each(ids, id =>
        notification.store.messageReceived(state.messages.get(id), state)
      )
    }

    let fakeNotification

    beforeEach(() => {
      window.Notification = sinon.spy(function create() {
        this.close = sinon.spy(() => {
          if (this.onclose) {
            this.onclose()
          }
        })
        fakeNotification = this
      })
      Notification.permission = 'granted'
      support.resetStore(notification.store)
    })

    it('should set supported', () => {
      assert.equal(notification.store.getInitialState().popupsSupported, true)
    })

    it('should set popupsPermission', () => {
      assert.equal(notification.store.getInitialState().popupsPermission, true)
    })

    describe('enabling popups', () => {
      it('should store enabled and reset pause time', () => {
        notification.store.enablePopups()
        sinon.assert.calledWithExactly(storage.set, 'notify', true)
        sinon.assert.calledWithExactly(storage.set, 'notifyPausedUntil', null)
      })
    })

    describe('disabling popups', () => {
      it('should store disabled and reset pause time', () => {
        notification.store.disablePopups()
        sinon.assert.calledWithExactly(storage.set, 'notify', false)
        sinon.assert.calledWithExactly(storage.set, 'notifyPausedUntil', null)
      })
    })

    describe('pausing popups for a time', () => {
      it('should store pause time', () => {
        const time = startTime + 1000
        notification.store.pausePopupsUntil(time)
        sinon.assert.calledWithExactly(storage.set, 'notifyPausedUntil', time)
      })
    })

    describe('when popup state changes in another tab', () => {
      beforeEach(() => {
        notification.store.chatStateChange({connected: true, joined: true})
        notification.store.onInactive()
        notification.store.storageChange(storageMock)
        simulateMessages(['id1'], mockChatState)
        clock.tick(0)
        sinon.assert.calledOnce(Notification)
      })

      it('should close popups if paused', () => {
        notification.store.storageChange(_.extend({}, storageMock, {
          notifyPausedUntil: Date.now() + 1000,
        }))
        sinon.assert.calledOnce(fakeNotification.close)
      })

      it('should close popups if disabled', () => {
        notification.store.storageChange(_.extend({}, storageMock, {
          notify: false,
        }))
        sinon.assert.calledOnce(fakeNotification.close)
      })
    })

    describe('setting a room notification mode', () => {
      it('should store mode', () => {
        notification.store.setRoomNotificationMode('ezzie', 'none')
        sinon.assert.calledWithExactly(storage.setRoom, 'ezzie', 'notifyMode', 'none')
      })
    })

    describe('when disconnected', () => {
      it('should set favicon', () => {
        notification.store.chatStateChange({connected: false, joined: false})
        sinon.assert.calledOnce(Heim.setFavicon)
        sinon.assert.calledWithExactly(Heim.setFavicon, notification.favicons.disconnected)
      })
    })

    describe('when popups enabled', () => {
      beforeEach(() => {
        notification.store.chatStateChange({connected: true, joined: true})
        Heim.setFavicon.reset()
        notification.store.onInactive()
        notification.store.storageChange(storageMock)
      })

      function checkNotify(opts) {
        it('should ' + (opts.expectFavicon ? '' : 'not ') + 'set favicon', () => {
          clock.tick(0)
          sinon.assert.calledOnce(Heim.setFavicon)
          sinon.assert.calledWithExactly(Heim.setFavicon, opts.expectFavicon ? opts.expectFavicon : '/static/favicon.png')
        })

        it('should set correct page title', () => {
          clock.tick(0)
          sinon.assert.calledOnce(Heim.setTitleMsg)
          sinon.assert.calledWithExactly(Heim.setTitleMsg, opts.expectTitleMsg)
        })

        it('should add notification', done => {
          support.listenOnce(notification.store, state => {
            _.each(opts.messageIds, messageId =>
              assert.equal(state.notifications.get(messageId), opts.expectKind)
            )
            done()
          })
          clock.tick(0)
        })

        it('should ' + (opts.expectPopupBody ? '' : 'not ') + 'display a popup', () => {
          clock.tick(0)
          if (opts.expectPopupBody) {
            sinon.assert.calledOnce(Notification)
            sinon.assert.calledWithExactly(Notification, 'ezzie', {
              icon: opts.expectPopupIcon,
              body: opts.expectPopupBody,
            })
          } else {
            sinon.assert.notCalled(Notification)
          }
        })
      }

      describe('before joined', () => {
        beforeEach(() => {
          notification.store.chatStateChange({connected: true, joined: false})
          Heim.setFavicon.reset()
        })

        describe('when page inactive', () => {
          describe('receiving logged messages', () => {
            beforeEach(() => {
              notification.store.messagesChanged([message2Reply1.id, message2Reply3.id], mockChatState2Reply3)
            })

            checkNotify({
              messageIds: [message2Reply1.id, message2Reply3.id],
              expectFavicon: notification.favicons.active,
              expectTitleMsg: 2,
              expectKind: 'new-reply',
            })
          })
        })

        describe('when page active', () => {
          beforeEach(() => {
            notification.store.onActive()
            Heim.setFavicon.reset()
            Heim.setTitleMsg.reset()
          })

          describe('receiving logged messages', () => {
            beforeEach(() => {
              notification.store.messagesChanged([message2Reply1.id, message2Reply3.id], mockChatState2Reply3)
            })

            checkNotify({
              messageIds: [message2Reply1.id, message2Reply3.id],
              expectFavicon: null,
              expectTitleMsg: '',
              expectKind: 'new-reply',
            })
          })
        })
      })

      describe('receiving a message while notifications paused', () => {
        it('should add a notification and not display a popup', done => {
          notification.store.storageChange(_.extend({}, storageMock, {notifyPausedUntil: startTime + 1000}))
          simulateMessages(['id1'], mockChatState)
          support.listenOnce(notification.store, state => {
            sinon.assert.notCalled(Notification)
            assert.equal(state.notifications.get('id1'), 'new-message')
            done()
          })
          clock.tick(0)
        })
      })

      function testNotifyModes(opts) {
        function test(expectPopup, mode) {
          describe('if notify mode is ' + JSON.stringify(mode) + '', () => {
            beforeEach(() => {
              notification.store.storageChange({notify: true, room: {ezzie: {notifyMode: mode}}})
              simulateMessages(opts.messageIds, opts.state)
            })

            checkNotify(_.assign({}, opts, {expectPopupBody: expectPopup && opts.expectPopupBody}))
          })
        }
        _.each(opts.popupModes, test)
        test(opts.popupModes.mention, undefined)
      }

      describe('receiving a message', () => {
        testNotifyModes({
          popupModes: {
            message: true,
            reply: false,
            mention: false,
            none: false,
          },
          messageIds: ['id1'],
          expectFavicon: notification.favicons.active,
          state: mockChatState,
          expectTitleMsg: 1,
          expectPopupIcon: notification.icons.active,
          expectPopupBody: 'logan: hello, ezzie!',
          expectKind: 'new-message',
        })
      })

      describe('receiving the same message again', () => {
        it('should not display a popup', () => {
          simulateMessages(['id1'], mockChatState)
          clock.tick(0)
          fakeNotification.close()
          simulateMessages(['id1'], mockChatState)
          clock.tick(0)
          sinon.assert.calledOnce(Notification)
          assert.equal(notification.store.state.notifications.get(message1.id), 'new-message')
        })
      })

      describe('when the same message has 2 changes triggered in the same tick', () => {
        it('should only increment the page title once', () => {
          simulateMessages(['id1'], mockChatState)
          notification.store.messagesChanged(['id1'], mockChatState)
          clock.tick(0)
          sinon.assert.calledOnce(Heim.setTitleMsg)
          sinon.assert.calledWithExactly(Heim.setTitleMsg, 1)
        })
      })

      describe('closing and receiving a new message', () => {
        it('should display a second popup', done => {
          simulateMessages(['id1'], mockChatState)
          clock.tick(0)
          fakeNotification.close()
          simulateMessages(['id2'], mockChatState2)
          support.listenOnce(notification.store, state => {
            sinon.assert.calledTwice(Notification)
            assert.equal(state.notifications.get(message1.id), 'new-message')
            done()
          })
          clock.tick(0)
        })
      })

      describe('receiving old messages', () => {
        it('should not replace existing newer notifications', done => {
          simulateMessages([message1.id, message2.id, message3.id], mockChatState3)
          clock.tick(0)
          simulateMessages([messageOld.id], mockChatState3Old)
          support.listenOnce(notification.store, state => {
            assert.equal(state.notifications.get(message1.id), 'new-message')
            assert.equal(state.notifications.get(message2.id), 'new-message')
            assert.equal(state.notifications.get(message3.id), 'new-message')
            done()
          })
          clock.tick(0)
        })
      })

      describe('receiving a child reply', () => {
        describe('when joined', () => {
          testNotifyModes({
            popupModes: {
              message: true,
              reply: true,
              mention: false,
              none: false,
            },
            messageIds: [message2Reply1.id],
            state: mockChatState2OwnReply,
            expectFavicon: notification.favicons.active,
            expectTitleMsg: 1,
            expectPopupIcon: notification.icons.active,
            expectPopupBody: 'logan: kitty?',
            expectKind: 'new-reply',
          })

          it('should replace an existing new-message notification', () => {
            simulateMessages(['id1'], mockChatState)
            clock.tick(0)
            assert.equal(notification.store.alerts['new-message'].messageId, message1.id)
            sinon.assert.calledOnce(Notification)
            const messageNotification = fakeNotification

            notification.store.messagesChanged([message2Reply1.id], mockChatState2OwnReply)
            clock.tick(0)
            assert.equal(notification.store.alerts['new-message'].messageId, message2Reply1.id)
            sinon.assert.calledOnce(messageNotification.close)
            sinon.assert.calledTwice(Notification)
          })
        })
      })

      describe('receiving a sibling reply', () => {
        describe('when joined', () => {
          testNotifyModes({
            popupModes: {
              message: true,
              reply: true,
              mention: false,
              none: false,
            },
            messageIds: [message2Reply3.id],
            state: mockChatState2Reply3,
            expectFavicon: notification.favicons.active,
            expectTitleMsg: 1,
            expectPopupIcon: notification.icons.active,
            expectPopupBody: 'kitty: mew?',
            expectKind: 'new-reply',
          })
        })
      })

      describe('receiving a mention', () => {
        describe('when joined', () => {
          testNotifyModes({
            popupModes: {
              message: true,
              reply: true,
              mention: true,
              none: false,
            },
            messageIds: [messageMention.id],
            state: mockChatStateMention,
            expectFavicon: notification.favicons.highlight,
            expectTitleMsg: 1,
            expectPopupIcon: notification.icons.highlight,
            expectPopupBody: 'tester: hey @ezzie!',
            expectKind: 'new-mention',
          })

          it('if seen before should not add a notification and not display a popup', () => {
            notification.store.messagesChanged([messageMention.id], mockChatStateMentionSeen)
            clock.tick(0)
            sinon.assert.notCalled(Notification)
            sinon.assert.notCalled(Heim.setFavicon)
            assert(!notification.store.state.notifications.has('id3'))
          })

          describe('and dismissing it and updating the mention again', () => {
            it('should not create another notification', done => {
              notification.store.messagesChanged([messageMention.id], mockChatStateMention)
              support.listenOnce(notification.store, state => {
                assert.equal(state.notifications.get(messageMention.id), 'new-mention')
                notification.store.dismissNotification(messageMention.id)
                clock.tick(0)
                assert(!notification.store.state.notifications.has(messageMention.id))
                notification.store.messagesChanged([messageMention.id], mockChatStateMention)
                clock.tick(0)
                assert(!notification.store.state.notifications.has(messageMention.id))
                done()
              })
              clock.tick(0)
            })
          })

          describe('and reconnecting later', () => {
            function testReset(expectHas, done, resetCallback) {
              notification.store.messagesChanged([messageMention.id], mockChatStateMention)
              support.listenOnce(notification.store, state => {
                assert.equal(state.notifications.get(messageMention.id), 'new-mention')
                sinon.assert.calledOnce(Notification)
                const mentionNotification = fakeNotification
                resetCallback()
                support.listenOnce(notification.store, state2 => {
                  assert.equal(state2.notifications.has(messageMention.id), expectHas)
                  sinon.assert.calledOnce(mentionNotification.close)
                  done()
                })
                clock.tick(0)
              })
              clock.tick(0)
            }

            describe('with the message no longer loaded', () => {
              it('should remove the notification and alert', done => {
                testReset(false, done, () => {
                  notification.store.messagesChanged(['__root'], emptyChatState)
                  notification.store.messagesChanged([message1.id], mockChatState)
                })
              })
            })

            describe('with the message as a shadow node', () => {
              it('should remove the notification and alert', done => {
                testReset(false, done, () => {
                  notification.store.messagesChanged(['__root'], mockChatStateMentionShadow)
                  notification.store.messagesChanged([message1.id], mockChatState)
                })
              })
            })

            describe('with the message re-loaded', () => {
              it('should remove the notification and alert', done => {
                testReset(true, done, () => {
                  notification.store.messagesChanged(['__root'], emptyChatState)
                  clock.tick(0)
                  notification.store.messagesChanged([messageMention.id], mockChatStateMention)
                })
              })
            })
          })

          describe('which is then deleted', () => {
            it('should remove the notification and alert', done => {
              notification.store.messagesChanged([messageMention.id], mockChatStateMention)
              support.listenOnce(notification.store, state => {
                assert.equal(state.notifications.get(messageMention.id), 'new-mention')
                sinon.assert.calledOnce(Notification)
                notification.store.messagesChanged([messageMention.id], mockChatStateMentionDeleted)
                support.listenOnce(notification.store, state2 => {
                  assert(!state2.notifications.has(messageMention.id))
                  sinon.assert.calledOnce(fakeNotification.close)
                  done()
                })
                clock.tick(0)
              })
              clock.tick(0)
            })
          })
        })

        describe('receiving multiple messages', () => {
          beforeEach(() => {
            simulateMessages([message2Reply1.id, message2Reply3.id], mockChatState2Reply3)
          })

          it('should display a popup for the latest message', done => {
            support.listenOnce(notification.store, () => {
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
      })

      describe('updating an orphaned message', () => {
        describe('when joined', () => {
          it('should not set favicon, add a notification, or display a popup', () => {
            notification.store.messagesChanged([message2Reply1.id], mockChatStateOrphan)
            clock.tick(0)
            sinon.assert.notCalled(Heim.setFavicon)
            assert(!notification.store.state.notifications.has(messageMention.id))
            sinon.assert.notCalled(Notification)
          })
        })
      })

      describe('removing an alert with mismatched message id', () => {
        it('should have no effect', () => {
          notification.store.messagesChanged([messageMention.id], mockChatStateMention)
          clock.tick(0)
          sinon.assert.calledOnce(Notification)
          notification.store.removeAlert('new-mention', 'nonexistent')
          sinon.assert.notCalled(fakeNotification.close)
        })
      })

      // workaround for Chrome behavior
      describe('closing an alert when the browser that doesn\'t call onclose consistently', () => {
        it('should try again in 500ms', () => {
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

        it('should not try again in 500ms if onclose was called', () => {
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

    describe('own messages', () => {
      beforeEach(() => {
        notification.store.chatStateChange({connected: true, joined: true})
        sinon.stub(notification, 'dismissNotification')
      })

      afterEach(() => {
        notification.dismissNotification.restore()
      })

      it('should dismiss notifications on parent and sibling messages', () => {
        simulateMessages(['id5'], mockChatState2Reply2Own)
        sinon.assert.calledWithExactly(notification.dismissNotification, message2.id)
        sinon.assert.calledWithExactly(notification.dismissNotification, message2Reply1.id)
      })

      it('should not dismiss notifications on top level messages (children of root)', () => {
        simulateMessages(['id2'], mockChatState2Own)
        sinon.assert.notCalled(notification.dismissNotification)
      })
    })

    describe('activity tracking', () => {
      beforeEach(() => {
        notification.store.chatStateChange({connected: true, joined: true})
        Heim.setFavicon.reset()
      })

      it('should start active', () => {
        assert.equal(notification.store.active, true)
      })

      it('should set active when page becomes active', () => {
        notification.store.onActive()
        assert.equal(notification.store.active, true)
      })

      it('should set inactive when page becomes inactive', () => {
        notification.store.onInactive()
        assert.equal(notification.store.active, false)
      })

      describe('when active and message received', () => {
        beforeEach(() => {
          notification.store.onActive()
          notification.store.storageChange(storageMock)
          simulateMessages(['id1'], mockChatState)
          Heim.setTitleMsg.reset()
          clock.tick(0)
        })

        it('should not open popups', () => {
          sinon.assert.notCalled(Notification)
        })

        it('should not alter page title', () => {
          sinon.assert.calledOnce(Heim.setTitleMsg)
          sinon.assert.calledWithExactly(Heim.setTitleMsg, '')
        })
      })

      describe('when inactive and message received', () => {
        beforeEach(() => {
          notification.store.onInactive()
          notification.store.storageChange(storageMock)
          simulateMessages(['id1'], mockChatState)
          clock.tick(0)
        })

        it('should close popup when window becomes active', () => {
          sinon.assert.calledOnce(Notification)
          notification.store.onActive()
          sinon.assert.calledOnce(fakeNotification.close)
        })

        it('should reset favicon when window becomes active', () => {
          sinon.assert.calledOnce(Heim.setFavicon)
          Heim.setFavicon.reset()

          notification.store.onActive()
          sinon.assert.calledOnce(Heim.setFavicon)
          sinon.assert.calledWithExactly(Heim.setFavicon, '/static/favicon.png')
        })

        it('should reset page title when window becomes active', () => {
          sinon.assert.calledOnce(Heim.setTitleMsg)
          Heim.setTitleMsg.reset()

          notification.store.onActive()
          sinon.assert.calledOnce(Heim.setTitleMsg)
          sinon.assert.calledWithExactly(Heim.setTitleMsg, '')
        })
      })
    })

    describe('with a popup showing', () => {
      beforeEach(() => {
        notification.store.chatStateChange({connected: true, joined: true})
        Heim.setFavicon.reset()
        notification.store.onInactive()
        notification.store.storageChange(storageMock)
        simulateMessages(['id1'], mockChatState)
        clock.tick(0)
        sinon.stub(ui, 'gotoMessageInPane')
        window.uiwindow = {focus: sinon.stub()}
      })

      afterEach(() => {
        ui.gotoMessageInPane.restore()
        delete window.uiwindow
      })

      it('should replace with another popup', () => {
        simulateMessages(['id2'], mockChatState2)
        clock.tick(0)
        sinon.assert.calledTwice(Notification)
      })

      it('should focus window and go to message when clicked', () => {
        fakeNotification.onclick()
        sinon.assert.calledOnce(window.uiwindow.focus)
        sinon.assert.calledOnce(ui.gotoMessageInPane)
        sinon.assert.calledWithExactly(ui.gotoMessageInPane, 'id1')
      })

      it('should close after 3 seconds', () => {
        clock.tick(3000)
        sinon.assert.calledOnce(fakeNotification.close)
      })

      it('should retain favicon state after timeout and reconnect', () => {
        clock.tick(3000)
        notification.store.chatStateChange({connected: false, joined: false})
        Heim.setFavicon.reset()
        notification.store.chatStateChange({connected: true, joined: true})
        sinon.assert.calledOnce(Heim.setFavicon)
        sinon.assert.calledWithExactly(Heim.setFavicon, notification.favicons.active)
      })

      describe('when window unloading', () => {
        it('should close the popup', () => {
          notification.store.removeAllAlerts()
          sinon.assert.calledOnce(fakeNotification.close)
        })
      })
    })

    describe('old seen notifications', () => {
      beforeEach(() => {
        notification.store.chatStateChange({connected: true, joined: true})
        notification.store.storageChange(storageMock)
      })

      function test(expectRemoved, done, seenTime, action) {
        const mockChatStateSeen = {
          roomName: 'ezzie',
          messages: new ChatTree().reset([
            message1,
          ]),
        }
        simulateMessages([message1.id], mockChatStateSeen)
        support.listenOnce(notification.store, state => {
          assert.equal(state.notifications.get(message1.id), 'new-message')
          mockChatStateSeen.messages.mergeNodes(message1.id, {_seen: Date.now()})
          notification.store.messagesChanged([message1.id], mockChatStateSeen)
          clock.tick(seenTime)
          action(mockChatStateSeen)
          support.listenOnce(notification.store, state2 => {
            assert.equal(expectRemoved, !state2.notifications.has(message1.id))
            done()
          })
          clock.tick(seenTime)
        })
        clock.tick(0)
      }

      it('should be removed when becoming inactive', done => {
        test(true, done, 40 * 1000, notification.store.onInactive)
      })

      it('should be removed if seen more than 30s ago when new messages come in', done => {
        test(true, done, 60 * 1000, state => {
          state.messages.add(message2)
          simulateMessages([message2.id], state)
        })
      })

      it('should not be removed if seen less than 30s ago', done => {
        test(false, done, 20 * 1000, state => {
          state.messages.add(message2)
          simulateMessages([message2.id], state)
        })
      })
    })

    describe('dismissing a nonexistent notification', () => {
      it('should have no effect', () => {
        assert(!notification.store.state.notifications.has('nonexistent'))
        notification.store.dismissNotification('nonexistent')
      })
    })

    describe('closing a nonexistent alert', () => {
      it('should have no effect', () => {
        assert(!_.has(notification.store.alerts, 'new-mention'))
        notification.store.closePopup('new-mention')
      })
    })
  })
})
