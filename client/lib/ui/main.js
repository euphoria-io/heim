var _ = require('lodash')
var React = require('react/addons')
var classNames = require('classnames')
var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup
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
var Panner = require('./panner')


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
    Reflux.listenTo(ui.panViewTo, 'panViewTo'),
  ],

  componentWillMount: function() {
    this._onResizeThrottled = _.throttle(this.onResize, 1000 / 30)
    Heim.addEventListener(uiwindow, 'resize', this._onResizeThrottled)
    this._threadScrollQueued = false
  },

  componentDidMount: function() {
    ui.focusEntry()
    this.onResize()
  },

  onResize: function() {
    ui.setUIMode({thin: uiwindow.innerWidth < 500})
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

  selectThread: function(id, itemEl) {
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

    ui.selectThread(id, itemEl)
  },

  dismissThreadPopup: function(ev) {
    if (!this.refs.threadList.getDOMNode().contains(ev.target)) {
      ui.deselectThread()
    }
  },

  onThreadSelect: function(ev, id) {
    if (this.state.ui.selectedThread == id && this.state.ui.threadPopupAnchorEl) {
      ui.deselectThread()
    } else {
      this.selectThread(id, ev.currentTarget)
    }
  },

  onThreadsScroll: function() {
    if (!this._threadScrollQueued && !this.state.ui.thin) {
      ui.deselectThread()
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
    this.selectThread(id, el)
  },

  panViewTo: function(x) {
    this.refs.panner.flingTo(x)
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
        var idx = _.indexOf(threadEls, threadListEl.querySelector('[data-thread-id="' + this.state.ui.selectedThread + '"]'))
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
        this.selectThread(threadEls[idx].dataset.threadId, threadEls[idx])
        return
      } else if (ev.key == 'Enter' && this.state.ui.focusedPane == this.state.ui.popupPane) {
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

  render: function() {
    var thin = this.state.ui.thin
    var selectedThread = this.state.ui.selectedThread

    var mainPaneThreadId
    if (thin && selectedThread) {
      mainPaneThreadId = 'thread-' + selectedThread
    }

    var threadPanes = this.state.ui.visiblePanes
      .filter((v, k) => /^thread-/.test(k))
      .toKeyedSeq()
      .map(paneId => this.state.ui.panes.get(paneId))
    var extraPanes = this.templateHook('thread-panes')

    var infoPaneHidden = this.state.ui.thin || !this.state.ui.infoPaneExpanded

    return (
      <Panner ref="panner" id="ui" snapPoints={infoPaneHidden ? {main: 0, info: 240} : {main: 0}} onMove={ui.onViewPan} className={classNames({'disconnected': this.state.chat.connected === false, 'info-pane-hidden': infoPaneHidden, 'info-pane-focused': this.state.ui.focusedPane == this.state.ui.popupPane})} onMouseDownCapture={this.onMouseDown} onClickCapture={this.onClick} onTouchMove={this.onTouchMove} onKeyDown={this.onKeyDown}>
        {this.state.storage && this.state.storage.useOpenDyslexic && <link rel="stylesheet" type="text/css" id="css" href="/static/od.css" />}
          {this.state.chat.authState && this.state.chat.authState != 'trying-stored' && <div className="hatch-shade fill" />}
        <div className="info-pane" onMouseEnter={ui.freezeInfo} onMouseLeave={ui.thawInfo}>
          <h2>discussions</h2>
          <div className="thread-list-container">
            <ThreadList ref="threadList" threadData={ui.store.threadData} threadTree={this.state.ui.frozenThreadList || this.state.chat.messages.threads} tree={this.state.chat.messages} onScroll={this.onThreadsScroll} onThreadSelect={this.onThreadSelect} />
          </div>
          <NotificationSettings roomName={this.state.chat.roomName} />
          <NotificationList tree={this.state.chat.messages} notifications={this.state.ui.frozenNotifications || this.state.notification.notifications} onNotificationSelect={this.onNotificationSelect} />
        </div>
        <div className="chat-pane-container main-pane">
          <ChatTopBar who={this.state.chat.who} roomName={this.state.chat.roomName} connected={this.state.chat.connected} joined={this.state.chat.joined} authType={this.state.chat.authType} updateReady={this.state.update.get('ready')} working={this.state.chat.loadingLogs} showSidebarButton={!this.state.ui.thin} sidebarExpanded={this.state.ui.infoPaneExpanded} collapseSidebar={ui.collapseInfoPane} expandSidebar={ui.expandInfoPane} />
          <div className="main-pane-stack">
            <ChatPane pane={this.state.ui.panes.get('main')} showTimeStamps={!this.state.ui.thin} onScrollbarSize={this.onScrollbarSize} disabled={!!mainPaneThreadId} />
            <ReactCSSTransitionGroup transitionName="slide" transitionLeave={!mainPaneThreadId} transitionEnter={false}>
              {mainPaneThreadId && <div key={mainPaneThreadId} className="main-pane-thread">
                <div className="top-bar">
                  <MessageText className="title" content={this.state.chat.messages.get(selectedThread).get('content')} />
                  <FastButton className="close" onClick={ui.deselectThread} />
                </div>
                <ChatPane key={mainPaneThreadId} pane={this.state.ui.panes.get(mainPaneThreadId)} showTimeStamps={!this.state.ui.thin} showParent={true} showAllReplies={true} onScrollbarSize={this.onScrollbarSize} />
              </div>}
            </ReactCSSTransitionGroup>
          </div>
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
        {!thin && <Bubble ref="threadPopup" className="thread-popup" anchorEl={this.state.ui.threadPopupAnchorEl} visible={!!this.state.ui.threadPopupAnchorEl} onDismiss={this.dismissThreadPopup} offset={() => ({ left: this.getDOMNode().getBoundingClientRect().left + 5, top: 26 })}>
          <div className="top-line">
            <FastButton className="to-pane" onClick={ui.popupToThreadPane}>new pane</FastButton>
            <FastButton className="scroll-to" onClick={ui.gotoPopupMessage}>go to</FastButton>
          </div>
          {selectedThread && <ChatPane key={this.state.ui.popupPane} pane={this.state.ui.panes.get(this.state.ui.popupPane)} afterMessagesRendered={() => this.refs.threadPopup.reposition()} showParent={true} showAllReplies={true} />}
        </Bubble>}
        {this.templateHook('page-bottom')}
      </Panner>
    )
  },
})
