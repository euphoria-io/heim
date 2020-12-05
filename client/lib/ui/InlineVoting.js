import React from 'react'
import Reflux from 'reflux'
import Immutable from 'immutable'

import actions from '../actions'
import chat from '../stores/chat'
import Tree from '../Tree'
import FastButton from './FastButton'
import MessageText from './MessageText'

export default React.createClass({
  displayName: 'InlineVoting',

  propTypes: {
    message: React.PropTypes.instanceOf(Immutable.Map).isRequired,
    tree: React.PropTypes.instanceOf(Tree).isRequired,
    className: React.PropTypes.string,
    title: React.PropTypes.string,
    style: React.PropTypes.string,
  },

  mixins: [
    Reflux.connect(chat.store, 'chat'),
  ],

  sendMessageIfPossible(text) {
    if (this.state.chat.joined && this.state.chat.nick) {
      actions.sendMessage(text, this.props.message.get('id'))
    }
  },

  upvote(evt) {
    this.sendMessageIfPossible('+1')
    if (evt) evt.stopPropagation()
  },

  downvote(evt) {
    this.sendMessageIfPossible('-1')
    if (evt) evt.stopPropagation()
  },

  render() {
    let upvotes = 0
    let downvotes = 0

    this.props.message.get('children').map(id => {
      const content = this.props.tree.get(id).get('content')

      if (/\s*\+1\s*/.test(content)) upvotes++
      if (/\s*-1\s*/.test(content)) downvotes++
    })

    const result = upvotes - downvotes
    const resultClass = (result > 0) ? 'approved' : (result < 0) ? 'rejected' : 'neutral' // eslint-disable-line

    const majorityPercent = Math.max(upvotes, downvotes) * 100 / (upvotes + downvotes)
    const percentText = ' (' + Math.round(majorityPercent) + '% ' + ((result > 0) ? '+' : '-') + ')'

    return (<span className="inline-voting">
      <FastButton onClick={this.upvote} className="approve">
        <MessageText content=":thumbsup:" onlyEmoji /> {upvotes}
      </FastButton>
      <FastButton onClick={this.downvote} className="disapprove">
        <MessageText content=":thumbsdown:" onlyEmoji /> {downvotes}
      </FastButton>
      <span className={resultClass}> {result}</span>
      {result !== 0 && <small>{percentText}</small>}
    </span>)
  },

})
