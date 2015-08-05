var _ = require('lodash')
var React = require('react')
var classNames = require('classnames')
var Immutable = require('immutable')
var moment = require('moment')

var ui = require('../stores/ui')
var FastButton = require('./fast-button')
var Embed = require('./embed')
var MessageText = require('./message-text')
var ChatEntry = require('./chat-entry')
var LiveTimeAgo = require('./live-time-ago')
var KeyboardActionHandler = require('./keyboard-action-handler')


var linearEasing = t => t
var snapEasing = t => (Math.pow(2.02 * t - 1.0303, 17) + t) / 3.5 + 0.475
var colorShouldStep = (x, last) => x - last > 0.01

var Message = module.exports = React.createClass({
  displayName: 'Message',

  mixins: [
    require('react-immutable-render-mixin'),
    require('./tree-node-mixin')(),
    require('./message-data-mixin')(props => props.pane.store.messageData, 'paneData'),
  ],

  statics: {
    visibleCount: 5,
    newFadeDuration: 60 * 1000,
  },

  getDefaultProps: function() {
    return {
      depth: 0,
      visibleCount: this.visibleCount - 1,
      showTimeStamps: false,
      showTimeAgo: false,
      showAllReplies: false,
      roomSettings: Immutable.Map(),
    }
  },

  focusMessage: function() {
    if (!uiwindow.getSelection().isCollapsed) {
      return
    }

    this.props.pane.toggleFocusMessage(this.props.nodeId, this.state.node.get('parent'))
  },

  render: function() {
    var message = this.state.node

    if (message.get('deleted')) {
      return <div data-message-id={message.get('id')} className="message-node deleted" />
    }

    var time = moment.unix(message.get('time'))
    if (this.props.nodeId == '__lastVisit') {
      return (
        <div className="line marker last-visit">
          <hr />
          <div className="label">last visit</div>
          <hr />
        </div>
      )
    }

    var children = message.get('children')
    var paneData = this.state.paneData
    var focused = paneData.get('focused')
    var contentExpanded = paneData.get('contentExpanded')
    var repliesExpanded = paneData.get('repliesExpanded') || this.props.showAllReplies
    var messagePane = message.get('_inPane')
    var repliesInOtherPane = messagePane && messagePane != this.props.pane.id
    var seen = message.get('_seen')

    this._sinceNew = Date.now() - time < Message.newFadeDuration
    var messageClasses = {
      'mention': message.get('_mention'),
      'unseen': !seen,
      'new': this._sinceNew,
    }

    var lineClasses = {
      'line': true,
      'expanded': contentExpanded,
    }

    var pane = this.props.pane
    var messageReplies
    var messageIndentedReplies
    if (repliesInOtherPane) {
      messageIndentedReplies = (
        <FastButton component="div" className={classNames('replies', 'in-pane', {'focus-target': focused})} onClick={this.focusOtherPane}>
          replies in pane <div className="pane-icon" />
          {focused && <div className="spacer"><button className="drag-handle" onClick={ev => ev.stopPropagation()} /></div>}
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
      var composingReply = focused && children.size === 0
      var inlineReplies = composingReply || this.props.visibleCount > 0 || this.props.showAllReplies
      var count, childCount, childNewCount
      if (!inlineReplies && !repliesExpanded) {
        count = this.props.tree.getCount(this.props.nodeId)
        childCount = count.get('descendants')
        childNewCount = count.get('newDescendants')
        messageIndentedReplies = (
          <div>
            <FastButton key="replies" component="div" className={classNames('replies', 'collapsed', {'focus-target': focused, 'empty': childCount === 0})} onClick={this.expandReplies}>
              {childCount === 0 ? 'reply'
                : childCount == 1 ? '1 reply'
                  : childCount + ' replies'}
              {childNewCount > 0 && <span className={classNames('new-count', {'new-mention': count.get('newMentionDescendants') > 0})}>{childNewCount}</span>}
              {childCount > 0 && <LiveTimeAgo className="ago" time={count.get('latestDescendantTime')} nowText="active" />}
              {<MessageText className="message-preview" content={this.props.tree.get(count.get('latestDescendant')).get('content').trim()} />}
              {focused && <div className="spacer"><button className="drag-handle" onClick={ev => ev.stopPropagation()} /></div>}
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
        var focusAction
        var expandRestOfReplies
        var canCollapse = !this.props.showAllReplies && children.size > this.props.visibleCount
        if (canCollapse && !repliesExpanded) {
          count = this.props.tree.calculateDescendantCount(this.props.nodeId, this.props.visibleCount)
          childCount = count.get('descendants')
          childNewCount = count.get('newDescendants')
          expandRestOfReplies = (
            <FastButton key="replies" component="div" className={classNames('expand-rest', {'focus-target': focused})} onClick={this.expandReplies}>
              {childCount} more
              {childNewCount > 0 && <span className={classNames('new-count', {'new-mention': count.get('newMentionDescendants') > 0})}>{childNewCount}</span>}
              <LiveTimeAgo className="ago" time={count.get('latestDescendantTime')} nowText="active" />
              {<MessageText className="message-preview" content={this.props.tree.get(count.get('latestDescendant')).get('content').trim()} />}
              {focused && <div className="spacer"><button className="drag-handle" onClick={ev => ev.stopPropagation()} /></div>}
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
        messageReplies = (
          <div ref="replies" className={classNames('replies', {'collapsible': canCollapse, 'expanded': canCollapse && repliesExpanded, 'inline': inlineReplies, 'empty': children.size === 0, 'focused': focused})}>
            <FastButton className="indent-line" onClick={canCollapse && (repliesExpanded ? this.collapseReplies : this.expandReplies)} empty={true} />
            <div className="content">
              {children.toIndexedSeq().map((nodeId, idx) =>
                <Message key={nodeId} pane={this.props.pane} tree={this.props.tree} nodeId={nodeId} depth={this.props.depth + 1} visibleCount={repliesExpanded ? Message.visibleCount : Math.floor((this.props.visibleCount - 1) / 2)} showTimeAgo={!expandRestOfReplies && idx == children.size - 1} showTimeStamps={this.props.showTimeStamps} roomSettings={this.props.roomSettings} />
              ).toArray()}
              {focusAction}
            </div>
          </div>
        )
      }
    }

    var content = message.get('content')
    var embeds = []
    content = content.replace(/(?:https?:\/\/)?(?:www\.|i\.|m\.)?imgur\.com\/(\w+)(\.\w+)?(\S*)/g, (match, id, ext, rest, offset, string) => {
      // jshint camelcase: false
      if (rest) {
        return string
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

    var messageAgo = (this.props.showTimeAgo || children.size >= 3) && <LiveTimeAgo className="ago" time={time} />

    var messageRender
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
      var action = contentExpanded ? 'collapse' : 'expand'
      var actionMethod = action + 'Content'
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

    var messageEmbeds
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
      <div data-message-id={message.get('id')} className={classNames('message-node', messageClasses)}>
        {this.props.showTimeStamps && <time ref="time" className="timestamp" dateTime={time.toISOString()} title={time.format('MMMM Do YYYY, h:mm:ss a')}>
          {time.format('h:mma')}
        </time>}
        <div className={classNames(lineClasses)} onClick={this.focusMessage}>
          <MessageText className="nick" onlyEmoji={true} style={{background: 'hsl(' + message.getIn(['sender', 'hue']) + ', 65%, 85%)'}} content={message.getIn(['sender', 'name'])} />
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

  componentDidMount: function() {
    var node = this.getDOMNode()

    if (node.classList.contains('new') && this._sinceNew < Message.newFadeDuration) {
      var lineEl = node.querySelector('.line')
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

  componentDidUpdate: function(prevProps, prevState) {
    if (this.state.node.get('_seen') && !prevState.node.get('_seen')) {
      var node = this.getDOMNode()

      var lineEl = node.querySelector('.line')
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
        var timestampEl = node.querySelector('.timestamp')
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

  afterRender: function() {
    if (this.refs.message && this.props.roomSettings.get('collapse') !== false) {
      var msgNode = this.refs.message.getDOMNode()
      if (msgNode.getBoundingClientRect().height > 200) {
        this.setState({contentTall: true})
      }
    }

    this.props.pane.messageRenderFinished()
  },

  expandContent: function(ev) {
    this.props.pane.setMessageData(this.props.nodeId, {contentExpanded: true})
    // don't focus the message
    ev.stopPropagation()
  },

  collapseContent: function(ev) {
    this.props.pane.setMessageData(this.props.nodeId, {contentExpanded: false})
    ev.stopPropagation()
  },

  expandReplies: function() {
    if (this.state.node.get('repliesExpanded')) {
      return
    }
    this.props.pane.setMessageData(this.props.nodeId, {repliesExpanded: true})
    if (this.state.paneData.get('focused')) {
      this.props.pane.focusEntry()
    }
  },

  collapseReplies: function() {
    this.props.pane.setMessageData(this.props.nodeId, {repliesExpanded: false})
  },

  freezeEmbed: function(idx) {
    this.refs['embed' + idx].freeze()
  },

  unfreezeEmbed: function(idx) {
    this.refs['embed' + idx].unfreeze()
  },

  openInPane: function() {
    ui.openThreadPane(this.props.nodeId)
  },

  focusOtherPane: function(ev) {
    ui.focusPane(this.state.node.get('_inPane'))
    ev.stopPropagation()
  },
})
