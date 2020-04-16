import _ from 'lodash'
import React from 'react'
import ReactDOM from 'react-dom'
import classNames from 'classnames'
import Immutable from 'immutable'
import moment from 'moment'

import ui, { Pane } from '../stores/ui'
import chat from '../stores/chat'
import Tree from '../Tree'
import FastButton from './FastButton'
import Embed from './Embed'
import MessageText from './MessageText'
import ChatEntry from './ChatEntry'
import LiveTimeAgo from './LiveTimeAgo'
import KeyboardActionHandler from './KeyboardActionHandler'
import EntryDragHandle from './EntryDragHandle'
import TreeNodeMixin from './TreeNodeMixin'
import MessageDataMixin from './MessageDataMixin'


const linearEasing = t => t
const snapEasing = t => (Math.pow(2.02 * t - 1.0303, 17) + t) / 3.5 + 0.475
const colorShouldStep = (x, last) => x - last > 0.01

const Message = React.createClass({
  displayName: 'Message',

  propTypes: {
    nodeId: React.PropTypes.string.isRequired,
    tree: React.PropTypes.instanceOf(Tree).isRequired,
    pane: React.PropTypes.instanceOf(Pane).isRequired,
    showTimeStamps: React.PropTypes.bool,
    showTimeAgo: React.PropTypes.bool,
    showAllReplies: React.PropTypes.bool,
    depth: React.PropTypes.number,
    visibleCount: React.PropTypes.number,
    maxDepth: React.PropTypes.number,
    roomSettings: React.PropTypes.instanceOf(Immutable.Map),
  },

  mixins: [
    require('react-immutable-render-mixin'),
    TreeNodeMixin(),
    MessageDataMixin(props => props.pane.store.messageData, 'paneData'),
  ],

  statics: {
    visibleCount: 5,
    maxDepth: 50,
    newFadeDuration: 60 * 1000,
  },

  getDefaultProps() {
    return {
      depth: 0,
      visibleCount: this.visibleCount - 1,
      maxDepth: this.maxDepth - 1,
      showTimeStamps: false,
      showTimeAgo: false,
      showAllReplies: false,
      roomSettings: Immutable.Map(),
    }
  },

  componentDidMount() {
    const node = ReactDOM.findDOMNode(this)

    if (node.classList.contains('new') && this._sinceNew < Message.newFadeDuration) {
      const lineEl = node.querySelector('.line')
      Heim.transition.add({
        startOffset: -this._sinceNew,
        step: x => {
          if (x < 1) {
            lineEl.style.background = 'rgba(0, 128, 0, ' + (1 - x) * 0.075 + ')'
          } else {
            lineEl.style.background = ''
          }
        },
        shouldStep: colorShouldStep,
        ease: linearEasing,
        duration: 60 * 1000,
        fps: 10,
      })
    }

    this.afterRender()
  },

  componentDidUpdate(prevProps, prevState) {
    if (this.state.node.get('_seen') && !prevState.node.get('_seen') && !this._hideSeen) {
      const node = ReactDOM.findDOMNode(this)

      const lineEl = node.querySelector('.line')
      Heim.transition.add({
        step: x => {
          if (x < 1) {
            lineEl.style.borderLeftColor = 'rgba(0, 128, 0, ' + (1 - x) * 0.75 + ')'
          } else {
            lineEl.style.borderLeftColor = ''
          }
        },
        shouldStep: colorShouldStep,
        ease: snapEasing,
        duration: 15 * 1000,
        fps: 30,
      })

      if (this.props.showTimeStamps) {
        const timestampEl = node.querySelector('.timestamp')
        Heim.transition.add({
          step: x => {
            if (x < 1) {
              timestampEl.style.color = 'rgb(170, ' + Math.round(170 + (241 - 170) * (1 - x)) + ', 170)'
            } else {
              timestampEl.style.color = ''
            }
          },
          shouldStep: colorShouldStep,
          ease: snapEasing,
          duration: 60 * 1000,
          fps: 30,
        })
      }
    }

    this.afterRender()
  },

  onClick() {
    if (ui.store.state.managerMode) {
      return
    }

    if (!uiwindow.getSelection().isCollapsed) {
      return
    }

    this.props.pane.toggleFocusMessage(this.props.nodeId, this.state.node.get('parent'))
  },

  onMouseDown(ev) {
    if (ui.store.state.managerMode) {
      const selected = this.state.node.get('_selected')
      chat.setMessageSelected(this.props.nodeId, !selected)
      ui.startToolboxSelectionDrag(!selected)
      ev.preventDefault()
    }
  },

  onMouseEnter() {
    if (ui.store.state.managerMode && ui.store.state.draggingToolboxSelection) {
      chat.setMessageSelected(this.props.nodeId, ui.store.state.draggingToolboxSelectionToggle)
    }
  },

  afterRender() {
    if (this.refs.message && this.props.roomSettings.get('collapse') !== false) {
      const msgNode = ReactDOM.findDOMNode(this.refs.message)
      if (msgNode.getBoundingClientRect().height > 200) {
        this.setState({contentTall: true})
      }
    }

    this.props.pane.messageRenderFinished()
  },

  expandContent(ev) {
    this.props.pane.setMessageData(this.props.nodeId, {contentExpanded: true})
    // don't focus the message
    ev.stopPropagation()
  },

  collapseContent(ev) {
    this.props.pane.setMessageData(this.props.nodeId, {contentExpanded: false})
    ev.stopPropagation()
  },

  expandReplies() {
    if (this.state.node.get('repliesExpanded')) {
      return
    }
    this.props.pane.setMessageData(this.props.nodeId, {repliesExpanded: true})
    if (this.state.paneData.get('focused')) {
      this.props.pane.focusEntry()
    }
  },

  collapseReplies() {
    this.props.pane.setMessageData(this.props.nodeId, {repliesExpanded: false})
  },

  freezeEmbed(idx) {
    this.refs['embed' + idx].freeze()
  },

  unfreezeEmbed(idx) {
    this.refs['embed' + idx].unfreeze()
  },

  openInPane() {
    ui.openThreadPane(this.props.nodeId)
  },

  focusOtherPane(ev) {
    ui.focusPane(this.state.node.get('_inPane'))
    ev.stopPropagation()
  },

  render() {
    const message = this.state.node
    const showAllReplies = this.props.showAllReplies || this.props.roomSettings.get('showAllReplies')

    if (message.get('deleted')) {
      return <div data-message-id={message.get('id')} className="message-node deleted" />
    }

    const time = moment.unix(message.get('time'))
    if (this.props.nodeId === '__lastVisit') {
      return (
        <div className="line marker last-visit">
          <hr />
          <div className="label">last visit</div>
          <hr />
        </div>
      )
    }

    let children = message.get('children')
    const paneData = this.state.paneData
    const count = this.props.tree.getCount(this.props.nodeId)
    const focused = paneData.get('focused')
    const contentExpanded = paneData.get('contentExpanded')

    let repliesExpanded
    if (this.props.maxDepth === 0) {
      repliesExpanded = paneData.get('repliesExpanded')
      if (repliesExpanded === null) {
        repliesExpanded = false
      }
    } else if (showAllReplies) {
      repliesExpanded = true
    } else {
      repliesExpanded = paneData.get('repliesExpanded')
      if (repliesExpanded === null) {
        repliesExpanded = count.get('ownDescendants') > 0
      }
    }

    const messagePane = message.get('_inPane')
    const repliesInOtherPane = messagePane && messagePane !== this.props.pane.id
    const seen = message.get('_seen')
    const mention = message.get('_mention')

    if (!chat.store.lastActive) {
      // hack: if the room hasn't been visited before (we're within the first
      // flush interval of the activity store), do not display seen indicators.
      // this prevents a wall of fading green lines on first visits.
      this._hideSeen = true
    }

    this._sinceNew = Date.now() - time < Message.newFadeDuration
    const messageClasses = {
      'mention': mention,
      'unseen': !seen && !this._hideSeen,
      'new': this._sinceNew,
      'selected': message.get('_selected'),
    }

    if (message.getIn(['sender', 'id']) === 'account:01c2c55f25zb4') {
      if (/^sponsored message:/.test(message.get('content'))) {
        messageClasses.sponsored = true
      } else {
        messageClasses.sponsorctl = true
      }
    }

    const lineClasses = {
      'line': true,
      'expanded': contentExpanded,
    }

    const pane = this.props.pane
    let messageReplies
    let messageIndentedReplies
    if (repliesInOtherPane) {
      messageIndentedReplies = (
        <FastButton component="div" className={classNames('replies', 'in-pane', {'focus-target': focused})} onClick={this.focusOtherPane}>
          replies in pane <div className="pane-icon" />
          {focused && <div className="spacer" onClick={ev => ev.stopPropagation()}><EntryDragHandle pane={this.props.pane} /></div>}
        </FastButton>
      )
      if (focused) {
        messageIndentedReplies = (
          <KeyboardActionHandler listenTo={pane.keydownOnPane} key="replies-key-handler" keys={{
            ArrowLeft: () => pane.moveMessageFocus('out'),
            ArrowRight: () => pane.moveMessageFocus('top'),
            ArrowUp: () => pane.moveMessageFocus('up'),
            ArrowDown: () => pane.moveMessageFocus('down'),
            Enter: this.focusOtherPane,
            Escape: () => pane.escape(),
          }}>
            {messageIndentedReplies}
          </KeyboardActionHandler>
        )
      }
    } else if (children.size > 0 || focused) {
      const composingReply = focused && children.size === 0
      const fullCollapse = this.props.visibleCount === 0 && !showAllReplies || this.props.maxDepth === 0
      const partialCollapse = children.size > this.props.visibleCount && !showAllReplies && !fullCollapse
      const inlineReplies = composingReply || !fullCollapse
      let childCount
      let childNewCount
      if (fullCollapse && !repliesExpanded && !composingReply) {
        childCount = count.get('descendants')
        childNewCount = count.get('newDescendants')
        let replyLabel
        if (childCount === 0) {
          replyLabel = 'reply'
        } else if (childCount === 1) {
          replyLabel = '1 reply'
        } else {
          replyLabel = childCount + ' replies'
        }
        messageIndentedReplies = (
          <div>
            <FastButton key="replies" component="div" className={classNames('replies', 'collapsed', {'focus-target': focused, 'empty': childCount === 0})} onClick={this.expandReplies}>
              {replyLabel}
              {childNewCount > 0 && <span className={classNames('new-count', {'new-mention': count.get('newMentionDescendants') > 0})}>{childNewCount}</span>}
              {childCount > 0 && <LiveTimeAgo className="ago" time={count.get('latestDescendantTime')} nowText="active" />}
              {<MessageText className="message-preview" content={this.props.tree.get(count.get('latestDescendant')).get('content').trim()} />}
              {focused && <div className="spacer" onClick={ev => ev.stopPropagation()}><EntryDragHandle pane={this.props.pane} /></div>}
            </FastButton>
          </div>
        )
        if (focused) {
          messageIndentedReplies = (
            <KeyboardActionHandler listenTo={pane.keydownOnPane} key="replies-key-handler" keys={{
              ArrowLeft: () => pane.moveMessageFocus('out'),
              ArrowRight: () => pane.moveMessageFocus('top'),
              ArrowUp: () => pane.moveMessageFocus('up'),
              ArrowDown: () => pane.moveMessageFocus('down'),
              Enter: this.expandReplies,
              TabEnter: this.openInPane,
              Escape: () => pane.escape(),
            }}>
              {messageIndentedReplies}
            </KeyboardActionHandler>
          )
        }
      } else {
        let focusAction
        let expandRestOfReplies
        const canCollapse = (fullCollapse || partialCollapse) && !composingReply
        if (partialCollapse && !repliesExpanded) {
          const descCount = this.props.tree.calculateDescendantCount(this.props.nodeId, this.props.visibleCount)
          childCount = descCount.get('descendants')
          childNewCount = descCount.get('newDescendants')
          expandRestOfReplies = (
            <FastButton key="replies" component="div" className={classNames('expand-rest', {'focus-target': focused})} onClick={this.expandReplies}>
              {childCount} more
              {childNewCount > 0 && <span className={classNames('new-count', {'new-mention': descCount.get('newMentionDescendants') > 0})}>{childNewCount}</span>}
              <LiveTimeAgo className="ago" time={descCount.get('latestDescendantTime')} nowText="active" />
              {<MessageText className="message-preview" content={this.props.tree.get(descCount.get('latestDescendant')).get('content').trim()} />}
              {focused && <div className="spacer" onClick={ev => ev.stopPropagation()}><EntryDragHandle pane={this.props.pane} /></div>}
            </FastButton>
          )
          if (focused) {
            expandRestOfReplies = (
              <KeyboardActionHandler listenTo={pane.keydownOnPane} key="replies-key-handler" keys={{
                ArrowLeft: () => pane.moveMessageFocus('out'),
                ArrowRight: () => pane.moveMessageFocus('top'),
                ArrowUp: () => pane.moveMessageFocus('up'),
                ArrowDown: () => pane.moveMessageFocus('down'),
                Enter: this.expandReplies,
                TabEnter: this.openInPane,
                Escape: () => pane.escape(),
              }}>
                {expandRestOfReplies}
              </KeyboardActionHandler>
            )
          }
          focusAction = expandRestOfReplies
          children = children.take(this.props.visibleCount)
        } else if (focused) {
          // expand replies on change so that another message coming in
          // (triggering expando) won't disrupt typing
          focusAction = <ChatEntry pane={pane} onChange={this.expandReplies} />
        }
        let collapseAction
        if (canCollapse) {
          collapseAction = repliesExpanded ? this.collapseReplies : this.expandReplies
        }
        messageReplies = (
          <div ref="replies" className={classNames('replies', {'collapsible': canCollapse, 'expanded': canCollapse && repliesExpanded, 'inline': inlineReplies, 'empty': children.size === 0, 'focused': focused})}>
            <FastButton className="indent-line" onClick={collapseAction} empty />
            <div className="content">
              {children.toIndexedSeq().map((nodeId, idx) =>
                <Message key={nodeId} pane={this.props.pane} tree={this.props.tree} nodeId={nodeId} depth={this.props.depth + 1} visibleCount={canCollapse && repliesExpanded || this.props.maxDepth === 0 ? Message.visibleCount : Math.floor((this.props.visibleCount - 1) / 2)} maxDepth={this.props.maxDepth === 0 ? Message.maxDepth - 1 : this.props.maxDepth - 1} showTimeAgo={!expandRestOfReplies && idx === children.size - 1} showTimeStamps={this.props.showTimeStamps} roomSettings={this.props.roomSettings} />
              )}
              {focusAction}
            </div>
          </div>
        )
      }
    }

    let content = message.get('content')
    const embeds = []
    content = content.replace(/(?:https?:\/\/)?(?:www\.|i\.|m\.)?imgur\.com\/(\w+)(\.\w+)?(\S*)/g, (match, id, ext, rest) => {
      if (rest) {
        return match
      }
      embeds.push({
        link: '//imgur.com/' + id,
        props: {
          kind: 'imgur',
          imgur_id: id,
        },
      })
      return ''
    })
    content = content.replace(/(?:https?:\/\/)?(imgs\.xkcd\.com\/comics\/.*\.(?:png|jpg)|i\.ytimg\.com\/.*\.jpg)/g, (match, imgUrl) => {
      embeds.push({
        link: '//' + imgUrl,
        props: {
          kind: 'img',
          url: '//' + imgUrl,
        },
      })
      return ''
    })
    content = _.trim(content)

    const messageAgo = (this.props.showTimeAgo || children.size >= 3 || mention) && <LiveTimeAgo className="ago" time={time} />

    let messageRender
    if (!content) {
      messageRender = null
    } else if (/^\/me/.test(content) && content.length < 240) {
      content = _.trim(content.replace(/^\/me ?/, ''))
      messageRender = (
        <div className="message">
          <MessageText content={content} className="message-emote" style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 95%)'}} />
          {messageAgo}
        </div>
      )
      lineClasses['line-emote'] = true
    } else if (this.state.contentTall && this.props.roomSettings.get('collapse') !== false) {
      const action = contentExpanded ? 'collapse' : 'expand'
      const actionMethod = action + 'Content'
      messageRender = (
        <div className="message-tall">
          <div className="message expando" onClick={this[actionMethod]}>
            <MessageText content={content} />
            <FastButton className="expand" onClick={this[actionMethod]}>{action}</FastButton>
          </div>
          {messageAgo}
        </div>
      )
    } else {
      messageRender = (
        <div className="message">
          <MessageText ref="message" content={content} />
          {messageAgo}
        </div>
      )
    }

    let messageEmbeds
    if (embeds.length) {
      messageEmbeds = (
        <div className="embeds">
          {_.map(embeds, (embed, idx) =>
            <a key={idx} href={embed.link} target="_blank" onMouseEnter={() => this.unfreezeEmbed(idx)} onMouseLeave={() => this.freezeEmbed(idx)}>
              <Embed ref={'embed' + idx} {...embed.props} />
            </a>
          )}
          {!messageRender && messageAgo}
        </div>
      )
      lineClasses['has-embed'] = true
    }

    return (
      <div data-message-id={message.get('id')} data-depth={this.props.depth} className={classNames('message-node', messageClasses)}>
        {this.props.showTimeStamps && <time ref="time" className="timestamp" dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')}>
          {time.format('h:mma')}
        </time>}
        <div className={classNames(lineClasses)} onClick={this.onClick} onMouseDown={this.onMouseDown} onMouseEnter={this.onMouseEnter}>
          <MessageText className="nick" onlyEmoji style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 85%)'}} content={message.getIn(['sender', 'name'])} />
          <span className="content">
            {messageRender}
            {messageEmbeds}
            {messageIndentedReplies}
          </span>
        </div>
        {messageReplies}
        {!focused && <div className="focus-anchor" data-message-id={message.get('id')} />}
      </div>
    )
  },
})

export default Message
