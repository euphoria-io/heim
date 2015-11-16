import _ from 'lodash'
import React from 'react'
import ReactDOM from 'react-dom'
import ReactCSSTransitionGroup from 'react-addons-css-transition-group'
import classNames from 'classnames'
import Reflux from 'reflux'

import chat from '../stores/chat'
import ui from '../stores/ui'
import update from '../stores/update'
import hueHash from '../hue-hash'
import notification from '../stores/notification'
import activity from '../stores/activity'
import HooksMixin from './hooks-mixin'
import ChatPane from './chat-pane'
import ChatTopBar from './chat-top-bar'
import MessageText from './message-text'
import NotificationSettings from './notification-settings'
import NotificationList from './notification-list'
import ThreadList from './thread-list'
import UserList from './user-list'
import AccountButton from './account-button'
import AccountAuthDialog from './account-auth-dialog'
import AccountSettingsDialog from './account-settings-dialog'
import Bubble from './bubble'
import FastButton from './fast-button'
import Panner from './panner'

export default React.createClass({
  displayName: 'Main',

  mixins: [
    HooksMixin,
    Reflux.ListenerMixin,
    Reflux.connect(chat.store, 'chat'),
    Reflux.connect(activity.store, 'activity'),
    Reflux.connect(ui.store, 'ui'),
    Reflux.connect(require('../stores/notification').store, 'notification'),
    Reflux.connect(update.store, 'update'),
    Reflux.connect(require('../stores/storage').store, 'storage'),
    Reflux.listenTo(ui.selectThreadInList, 'selectThreadInList'),
    Reflux.listenTo(ui.panViewTo, 'panViewTo'),
    Reflux.listenTo(ui.tabKeyCombo, 'onTabKeyCombo'),
  ],

  componentWillMount() {
    this._onResizeThrottled = _.throttle(this.onResize, 1000 / 30)
    Heim.addEventListener(uiwindow, 'resize', this._onResizeThrottled)
    this._threadScrollQueued = false
    this.onResize()

    this.listenTo(ui.globalMouseUp, 'globalMouseUp')
  },

  componentDidMount() {
    ui.focusEntry()
  },

  onResize() {
    ui.setUISize(uiwindow.innerWidth, uiwindow.innerHeight)
  },

  onScrollbarSize(width) {
    this.setState({scrollbarWidth: width})
  },

  onMouseDown() {
    // FIXME: preventing/canceling a mousedown in React doesn't seem to stop
    // the subsequent click event, so we have to resort to this hack.
    this._isFocusClick = Date.now() - this.state.activity.focusChangedAt < 100
  },

  onClick(ev) {
    if (this._ignoreClick(ev)) {
      return
    }

    // prevent clicks to focus window and link clicks from triggering elements
    if (this._isFocusClick || ev.target.nodeName === 'A') {
      ev.stopPropagation()
    }

    if (this._isFocusClick) {
      ui.focusEntry()
    }
  },

  onPaneClick(paneId, ev) {
    if (this._ignoreClick(ev)) {
      return
    }

    if ((this.state.ui.thin || paneId === 'main') && this.state.ui.panPos !== paneId) {
      ui.focusPane(paneId)
      ui.panViewTo(paneId)
      ev.stopPropagation()
    }
  },

  onTouchMove(ev) {
    // prevent inertial scrolling of non-scrollable elements in Mobile Safari
    if (Heim.isiOS) {
      let el = ev.target
      while (el && el !== uidocument.body) {
        if (el.classList.contains('top-bar')) {
          ev.preventDefault()
          return
        }

        if (el.clientHeight < el.scrollHeight) {
          return
        }
        el = el.parentNode
      }
      ev.preventDefault()
    }
  },

  onTabKeyCombo(ev) {
    if (ev.key === 'ArrowLeft') {
      ui.focusLeftPane()
      return
    } else if (ev.key === 'ArrowRight') {
      ui.focusRightPane()
      return
    } else if (ev.key === 'ArrowUp' || ev.key === 'ArrowDown') {
      if (!this.state.ui.threadPopupAnchorEl) {
        return
      }

      ev.preventDefault()

      const threadListEl = ReactDOM.findDOMNode(this.refs.threadList)
      const threadEls = threadListEl.querySelectorAll('[data-thread-id]')
      let idx = _.indexOf(threadEls, threadListEl.querySelector('[data-thread-id="' + this.state.ui.selectedThread + '"]'))
      if (idx === -1) {
        throw new Error('could not locate current thread in list')
      }

      if (ev.key === 'ArrowUp') {
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
    } else if (ev.key === 'Enter' && this.state.ui.focusedPane === this.state.ui.popupPane) {
      ui.popupToThreadPane()
      return
    } else if (ev.key === 'Backspace') {
      if (/^thread-/.test(this.state.ui.focusedPane)) {
        ui.closeFocusedThreadPane()
        return
      }
    } else if (uiwindow.getSelection().isCollapsed) {
      ui.focusEntry()
    }
  },

  onThreadSelect(ev, id) {
    if (ev.button === 1) {
      ui.openThreadPane(id)
    } else if (this.state.ui.selectedThread === id && this.state.ui.threadPopupAnchorEl) {
      ui.deselectThread()
    } else {
      this.selectThread(id, ev.currentTarget)
    }
  },

  onThreadsScroll() {
    if (!this._threadScrollQueued && !this.state.ui.thin) {
      ui.deselectThread()
    }
    this._threadScrollQueued = false
  },

  onNotificationSelect(ev, id) {
    notification.dismissNotification(id)
    ui.gotoMessageInPane(id)
  },

  _ignoreClick(ev) {
    return !uiwindow.getSelection().isCollapsed || ev.target.nodeName === 'BUTTON'
  },

  selectThread(id, itemEl) {
    // poor man's scrollIntoViewIfNeeded
    const parentEl = ReactDOM.findDOMNode(this.refs.threadList)
    const itemBox = itemEl.getBoundingClientRect()
    const parentBox = parentEl.getBoundingClientRect()
    if (itemBox.top < parentBox.top) {
      this._threadScrollQueued = true
      itemEl.scrollIntoView(true)
    } else if (itemBox.bottom > parentBox.bottom) {
      this._threadScrollQueued = true
      itemEl.scrollIntoView(false)
    }

    ui.selectThread(id, itemEl)
  },

  dismissThreadPopup(ev) {
    if (!ReactDOM.findDOMNode(this.refs.threadList).contains(ev.target)) {
      ui.deselectThread()
    }
  },

  selectThreadInList(id) {
    const threadListEl = ReactDOM.findDOMNode(this.refs.threadList)
    let el = threadListEl.querySelector('[data-thread-id="' + id + '"]')
    let targetId = id
    if (!el) {
      el = threadListEl.querySelector('[data-thread-id]')
      targetId = el.dataset.threadId
    }
    this.selectThread(targetId, el)
  },

  panViewTo(x) {
    this.refs.panner.flingTo(x)
  },

  openManagerToolbox() {
    ui.openManagerToolbox(ReactDOM.findDOMNode(this.refs.toolboxButton))
  },

  globalMouseUp() {
    if (this.state.ui.draggingMessageSelection) {
      ui.finishMessageSelectionDrag()
    }
  },

  render() {
    const thin = this.state.ui.thin
    const selectedThread = this.state.ui.selectedThread

    let mainPaneThreadId
    if (thin && selectedThread) {
      mainPaneThreadId = 'thread-' + selectedThread
    }

    const threadPanes = this.state.ui.visiblePanes
      .filter((v, k) => /^thread-/.test(k))
      .toKeyedSeq()
      .map(paneId => this.state.ui.panes.get(paneId))
    const extraPanes = this.templateHook('thread-panes')
    const threadPanesFlex = threadPanes.size + extraPanes.length

    const infoPaneHidden = thin || !this.state.ui.infoPaneExpanded
    const infoPaneOpen = infoPaneHidden ? this.state.ui.panPos === 'info' : this.state.ui.infoPaneExpanded
    const sidebarPaneHidden = thin

    const roomName = this.state.chat.roomName

    const snapPoints = {main: 0}
    if (infoPaneHidden) {
      snapPoints.info = 240
    }
    if (sidebarPaneHidden) {
      snapPoints.sidebar = -150
    }

    const selectedMessageCount = this.state.chat.selectedMessages.size
    // lazy load manager toolbox ui (and store)
    const ManagerToolbox = this.state.ui.managerMode && require('./manager-toolbox').default

    return (
      <div id="ui" className={classNames({'disconnected': this.state.chat.connected === false})} onKeyDown={ui.keydownOnPage}>
        <Panner ref="panner" id="ui-panes" snapPoints={snapPoints} onMove={ui.onViewPan} className={classNames({'info-pane-hidden': infoPaneHidden, 'sidebar-pane-hidden': sidebarPaneHidden, 'info-pane-focused': this.state.ui.focusedPane === this.state.ui.popupPane, 'manager-mode': this.state.ui.managerMode})} onMouseDownCapture={this.onMouseDown} onClickCapture={this.onClick} onTouchMove={this.onTouchMove}>
          {this.state.storage && this.state.storage.useOpenDyslexic && <link rel="stylesheet" type="text/css" id="css" href="/static/od.css" />}
          <div className="info-pane" onMouseEnter={ui.freezeInfo} onMouseLeave={ui.thawInfo}>
            {this.state.ui.managerMode && <FastButton ref="toolboxButton" className={classNames('toolbox-button', {'empty': !this.state.chat.selectedMessages.size, 'selected': !!this.state.ui.managerToolboxAnchorEl})} onClick={this.state.ui.managerToolboxAnchorEl ? ui.closeManagerToolbox : this.openManagerToolbox}>toolbox {selectedMessageCount > -1 && <span className="count">{selectedMessageCount} selected</span>}</FastButton>}
            {this.state.chat.connected && <div className="account-area"><AccountButton ref="accountButton" account={this.state.chat.account} onOpenAccountAuthDialog={ui.openAccountAuthDialog} onOpenAccountSettingsDialog={ui.openAccountSettingsDialog} /></div>}
            <h2>discussions</h2>
            <div className="thread-list-container">
              <ThreadList ref="threadList" threadData={ui.store.threadData} threadTree={this.state.ui.frozenThreadList || this.state.chat.messages.threads} tree={this.state.chat.messages} onScroll={this.onThreadsScroll} onThreadSelect={this.onThreadSelect} />
            </div>
            {!(this.state.ui.thin && Heim.isTouch) && <NotificationSettings roomName={roomName} />}
            <NotificationList tree={this.state.chat.messages} notifications={this.state.ui.frozenNotifications || this.state.notification.notifications} onNotificationSelect={this.onNotificationSelect} animate={!this.state.ui.thin} />
          </div>
          <div className="chat-pane-container main-pane" onClickCapture={_.partial(this.onPaneClick, 'main')}>
            <ChatTopBar who={this.state.chat.who} roomName={roomName} connected={this.state.chat.connected} joined={!!this.state.chat.joined} authType={this.state.chat.authType} isManager={this.state.chat.isManager} managerMode={this.state.ui.managerMode} working={this.state.chat.loadingLogs} showInfoPaneButton={!thin || !Heim.isTouch} infoPaneOpen={infoPaneOpen} collapseInfoPane={ui.collapseInfoPane} expandInfoPane={ui.expandInfoPane} toggleUserList={ui.toggleUserList} toggleManagerMode={ui.toggleManagerMode} />
            {this.templateHook('main-pane-top')}
            <ReactCSSTransitionGroup className="notice-stack" transitionName="slide-down" transitionEnterTimeout={150} transitionLeaveTimeout={150}>
              {this.state.ui.notices.contains('notifications') && this.state.notification.popupsSupported && <div className="notice notifications">
                <div className="content">
                  <span className="title">what would you like notifications for?</span>
                  <div className="actions">
                    <FastButton onClick={() => ui.notificationsNoticeChoice('message')}>new messages</FastButton>
                    or
                    <FastButton onClick={() => ui.notificationsNoticeChoice('mention')}>just mentions<span className="long"> of @{hueHash.normalize(this.state.chat.nick)}</span></FastButton>
                  </div>
                </div>
                <FastButton className="close" onClick={() => ui.dismissNotice('notifications')} />
              </div>}
              {this.state.update.get('ready') && <FastButton className="update-button" onClick={update.perform}><p>update ready<em>{Heim.isTouch ? 'tap' : 'click'} to reload</em></p></FastButton>}
            </ReactCSSTransitionGroup>
            <div className="main-pane-stack">
              <ChatPane pane={this.state.ui.panes.get('main')} showTimeStamps={this.state.ui.showTimestamps} onScrollbarSize={this.onScrollbarSize} disabled={!!mainPaneThreadId} />
              <ReactCSSTransitionGroup transitionName="slide" transitionLeave={!mainPaneThreadId} transitionLeaveTimeout={200} transitionEnter={false}>
                {mainPaneThreadId && <div key={mainPaneThreadId} className="main-pane-cover main-pane-thread">
                  <div className="top-bar">
                    <MessageText className="title" content={this.state.chat.messages.get(selectedThread).get('content')} />
                    <FastButton className="close" onClick={ui.deselectThread} />
                  </div>
                  <ChatPane key={mainPaneThreadId} pane={this.state.ui.panes.get(mainPaneThreadId)} showTimeStamps={this.state.ui.showTimestamps} showParent showAllReplies onScrollbarSize={this.onScrollbarSize} />
                </div>}
                {thin && this.state.ui.managerToolboxAnchorEl && <div key="manager-toolbox" className="main-pane-cover">
                  <ManagerToolbox />
                </div>}
              </ReactCSSTransitionGroup>
            </div>
          </div>
          {(thin || this.state.ui.sidebarPaneExpanded) && <div className="sidebar-pane">
            <UserList users={this.state.chat.who} />
            {this.templateHook('main-sidebar')}
          </div>}
          {!thin && <div className="thread-panes" style={{flex: threadPanesFlex, WebkitFlex: threadPanesFlex}}>
            {extraPanes}
            {threadPanes.entrySeq().map(([paneId, pane], idx) => {
              const threadId = paneId.substr('thread-'.length)
              const title = this.state.chat.messages.get(threadId).get('content')
              return (
                <div key={paneId} className="chat-pane-container" style={{zIndex: threadPanes.size - idx}} onClickCapture={_.partial(this.onPaneClick, paneId)}>
                  <div className="top-bar">
                    <MessageText className="title" content={title} title={title} />
                    <FastButton className="close" onClick={_.partial(ui.closeThreadPane, threadId)} />
                  </div>
                  <ChatPane pane={pane} showParent showAllReplies />
                </div>
              )
            })}
          </div>}
          {!thin && <Bubble ref="threadPopup" className="thread-popup bubble-from-left" anchorEl={this.state.ui.threadPopupAnchorEl} visible={!!this.state.ui.threadPopupAnchorEl} onDismiss={this.dismissThreadPopup} offset={() => ({ left: ReactDOM.findDOMNode(this).getBoundingClientRect().left + 5, top: 26 })}>
            <div className="top-line">
              <FastButton className="to-pane" onClick={ui.popupToThreadPane}>new pane</FastButton>
              <FastButton className="scroll-to" onClick={ui.gotoPopupMessage}>go to</FastButton>
            </div>
            {selectedThread && <ChatPane key={this.state.ui.popupPane} pane={this.state.ui.panes.get(this.state.ui.popupPane)} afterRender={() => this.refs.threadPopup.reposition()} showParent showAllReplies />}
          </Bubble>}
          {!thin && this.state.ui.managerMode && <Bubble ref="managerToolboxPopup" className="manager-toolbox-popup bubble-from-top" anchorEl={this.state.ui.managerToolboxAnchorEl} visible={!!this.state.ui.managerToolboxAnchorEl} offset={anchorBox => ({ left: anchorBox.width, top: -anchorBox.height })}>
            <ManagerToolbox />
          </Bubble>}
          {this.templateHook('page-bottom')}
        </Panner>
        {this.state.ui.modalDialog === 'account-auth' && <AccountAuthDialog onClose={ui.closeDialog} />}
        {this.state.ui.modalDialog === 'account-settings' && <AccountSettingsDialog onClose={ui.closeDialog} />}
      </div>
    )
  },
})
