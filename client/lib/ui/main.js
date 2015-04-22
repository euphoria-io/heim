var _ = require('lodash')
var React = require('react')
var classNames = require('classnames')
var Reflux = require('reflux')

var chat = require('../stores/chat')
var ui = require('../stores/ui')
var notification = require('../stores/notification')
var activity = require('../stores/activity')
var ChatPane = require('./chat-pane')
var ChatTopBar = require('./chat-top-bar')
var MessageText = require('./message-text')
var NotificationSettings = require('./notification-settings')
var NotificationList = require('./notification-list')
var ThreadList = require('./thread-list')
var Bubble = require('./bubble')
var FastButton = require('./fast-button')


module.exports = React.createClass({
  displayName: 'Main',

  mixins: [
    require('./hooks-mixin'),
    Reflux.ListenerMixin,
    Reflux.connect(chat.store, 'chat'),
    Reflux.connect(activity.store, 'activity'),
    Reflux.connect(ui.store, 'ui'),
    Reflux.connect(require('../stores/notification').store, 'notification'),
    Reflux.connect(require('../stores/update').store, 'update'),
    Reflux.connect(require('../stores/storage').store, 'storage'),
    Reflux.listenTo(ui.selectThreadInList, 'selectThreadInList'),
  ],

  componentWillMount: function() {
    Heim.addEventListener(uiwindow, 'resize', this.onResize)
    this.listenTo(this.state.ui.panes.get('popup').afterMessagesRendered, this.afterPopupMessagesRendered)
    this._threadScrollQueued = false
  },

  componentDidMount: function() {
    ui.focusEntry()
  },

  onResize: function(width) {
    // TODO
    //
    // this.props.onResize(node.offsetWidth, node.offsetHeight)
    //
    // update scrollers on resize too
    return
    this.setState({
      thin: width < 500,
      wide: width > 920,
    })
  },

  onScrollbarSize: function(width) {
    this.setState({scrollbarWidth: width})
  },

  onMouseDown: function() {
    // FIXME: preventing/canceling a mousedown in React doesn't seem to stop
    // the subsequent click event, so we have to resort to this hack.
    this._isFocusClick = Date.now() - this.state.activity.focusChangedAt < 100
  },

  onClick: function(ev) {
    if (!uiwindow.getSelection().isCollapsed || ev.target.nodeName == 'BUTTON') {
      return
    }

    // prevent clicks to focus window and link clicks from triggering elements
    if (this._isFocusClick || ev.target.nodeName == 'A') {
      ev.stopPropagation()
    }

    if (this._isFocusClick) {
      ui.focusEntry()
    }
  },

  onTouchMove: function(ev) {
    // prevent inertial scrolling of the top level in Mobile Safari
    if (Heim.isiOS && !this.refs.scroller.getDOMNode().contains(ev.target)) {
      ev.preventDefault()
    }
  },

  showThreadPopup: function(id, itemEl) {
    // poor man's scrollIntoViewIfNeeded
    var parentEl = this.refs.threadList.getDOMNode()
    var itemBox = itemEl.getBoundingClientRect()
    var parentBox = parentEl.getBoundingClientRect()
    if (itemBox.top < parentBox.top) {
      this._threadScrollQueued = true
      itemEl.scrollIntoView(true)
    } else if (itemBox.bottom > parentBox.bottom) {
      this._threadScrollQueued = true
      itemEl.scrollIntoView(false)
    }

    ui.showThreadPopup(id, itemEl)
  },

  dismissThreadPopup: function(ev) {
    if (!this.refs.threadList.getDOMNode().contains(ev.target)) {
      ui.hideThreadPopup()
    }
  },

  onThreadSelect: function(ev, id) {
    if (this.state.ui.threadPopupRoot == id && this.state.ui.threadPopupAnchorEl) {
      ui.hideThreadPopup()
    } else {
      this.showThreadPopup(id, ev.currentTarget)
    }
  },

  onThreadsScroll: function() {
    if (!this._threadScrollQueued) {
      ui.hideThreadPopup()
    }
    this._threadScrollQueued = false
  },

  selectThreadInList: function(id) {
    var threadListEl = this.refs.threadList.getDOMNode()
    var el = threadListEl.querySelector('[data-thread-id="' + id + '"]')
    if (!el) {
      el = threadListEl.querySelector('[data-thread-id]')
      id = el.dataset.threadId
    }
    this.showThreadPopup(id, el)
  },

  onNotificationSelect: function(ev, id) {
    notification.dismissNotification(id)
    ui.gotoMessageInPane(id)
  },

  onKeyDown: function(ev) {
    if (Heim.tabPressed) {
      if (ev.key == 'ArrowLeft') {
        ui.focusLeftPane()
        return
      } else if (ev.key == 'ArrowRight') {
        ui.focusRightPane()
        return
      } else if (ev.key == 'ArrowUp' || ev.key == 'ArrowDown') {
        if (!this.state.ui.threadPopupAnchorEl) {
          return
        }

        ev.preventDefault()

        var threadListEl = this.refs.threadList.getDOMNode()
        var threadEls = threadListEl.querySelectorAll('[data-thread-id]')
        var idx = _.indexOf(threadEls, threadListEl.querySelector('[data-thread-id="' + this.state.ui.threadPopupRoot + '"]'))
        if (idx == -1) {
          throw new Error('could not locate current thread in list')
        }

        if (ev.key == 'ArrowUp') {
          if (idx === 0) {
            return
          }
          idx--
        } else {
          if (idx >= threadEls.length - 1) {
            return
          }
          idx++
        }
        this.showThreadPopup(threadEls[idx].dataset.threadId, threadEls[idx])
        return
      } else if (ev.key == 'Enter' && this.state.ui.focusedPane == 'popup') {
        ui.popupToThreadPane()
        return
      } else if (ev.key == 'Backspace') {
        if (/^thread-/.test(this.state.ui.focusedPane)) {
          ui.closeFocusedThreadPane()
          return
        }
      }
    } else if (uiwindow.getSelection().isCollapsed) {
      ui.focusEntry()
    }

    ui.keydownOnPage(ev)
  },

  afterPopupMessagesRendered: function() {
    this.refs.threadPopup.reposition()
  },

  render: function() {
    var threadPanes = this.state.ui.panes.filter((v, k) => /^thread-/.test(k))
    var extraPanes = this.templateHook('thread-panes')
    return (
      <div id="ui" className={classNames({'disconnected': this.state.chat.connected === false})} onMouseDownCapture={this.onMouseDown} onClickCapture={this.onClick} onTouchMove={this.onTouchMove} onKeyDown={this.onKeyDown}>
        {this.state.storage && this.state.storage.useOpenDyslexic && <link rel="stylesheet" type="text/css" id="css" href="/static/od.css" />}
          {this.state.chat.authState && this.state.chat.authState != 'trying-stored' && <div className="hatch-shade fill" />}
        <div className="info-pane" onMouseEnter={ui.freezeInfo} onMouseLeave={ui.thawInfo}>
          <h2>discussions</h2>
          <div className="thread-list-container">
            <ThreadList ref="threadList" pane={this.state.ui.panes.get('popup')} threadTree={this.state.ui.frozenThreadList || this.state.chat.messages.threads} tree={this.state.chat.messages} onScroll={this.onThreadsScroll} onThreadSelect={this.onThreadSelect} />
          </div>
          <NotificationSettings roomName={this.state.chat.roomName} />
          <NotificationList tree={this.state.chat.messages} pane={this.state.ui.panes.get('popup')} notifications={this.state.ui.frozenNotifications || this.state.notification.notifications} onNotificationSelect={this.onNotificationSelect} />
        </div>
        <div className="chat-pane-container main-pane">
          <ChatTopBar who={this.state.chat.who} roomName={this.state.chat.roomName} connected={this.state.chat.connected} joined={this.state.chat.joined} authType={this.state.chat.authType} updateReady={this.state.update.get('ready')} working={this.state.chat.loadingLogs} />
          <ChatPane pane={this.state.ui.panes.get('main')} showTimeStamps={true} onScrollbarSize={this.onScrollbarSize} />
          <div className="sidebar" style={{marginRight: this.state.scrollbarWidth}}>
            {this.templateHook('main-sidebar')}
          </div>
        </div>
        <div className="thread-panes" style={{flex: threadPanes.size + extraPanes.length}}>
          {extraPanes}
          {threadPanes.entrySeq().map(([paneId, pane], idx) => {
            var threadId = paneId.substr('thread-'.length)
            return (
              <div key={paneId} className="chat-pane-container" style={{zIndex: threadPanes.size - idx}}>
                <div className="top-bar">
                  <MessageText className="title" content={this.state.chat.messages.get(threadId).get('content')} />
                  <FastButton className="close" onClick={_.partial(ui.closeThreadPane, threadId)} />
                </div>
                <ChatPane pane={pane} showParent={true} showAllReplies={true} />
              </div>
            )
          }).toArray()}
        </div>
        <Bubble ref="threadPopup" className="thread-popup" anchorEl={this.state.ui.threadPopupAnchorEl} visible={!!this.state.ui.threadPopupAnchorEl} onDismiss={this.dismissThreadPopup} topOffset={-26}>
          <div className="top-line">
            <FastButton className="to-pane" onClick={ui.popupToThreadPane}>new pane</FastButton>
            <FastButton className="scroll-to" onClick={ui.gotoPopupMessage}>go to</FastButton>
          </div>
          <ChatPane pane={this.state.ui.panes.get('popup')} showParent={true} showAllReplies={true} />
        </Bubble>
        {this.templateHook('page-bottom')}
      </div>
    )
  },
})
