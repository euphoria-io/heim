import _ from 'lodash'
import React from 'react'
import Immutable from 'immutable'
import twemoji from 'twemoji'

import actions from '../actions'
import Tree from '../Tree'
import FastButton from './FastButton'
import Message from './Message'
import MessageText from './MessageText'

export default React.createClass({
  displayName: 'InlineVoting',

  propTypes: {
    message: React.PropTypes.instanceOf(Immutable.Map).isRequired,
    tree: React.PropTypes.instanceOf(Tree).isRequired,
    className: React.PropTypes.string,
    title: React.PropTypes.string,
    style: React.PropTypes.string
  },

  upvote(evt) {
    actions.sendMessage('+1', this.props.message.get('id'))
    if (evt) evt.stopPropagation();
  },

  downvote(evt) {
    actions.sendMessage('-1', this.props.message.get('id'))
    if (evt) evt.stopPropagation();
  },

  render() {
    let upvotes = 0, downvotes = 0

    this.props.message.get('children').map(id => {
      const content = this.props.tree.get(id).get('content')

      if (/\s*\+1\s*/.test(content)) upvotes++
      if (/\s*-1\s*/.test(content)) downvotes++
    })

    const result = upvotes - downvotes;
    const resultClass = (result > 0) ? "approved" : (result < 0) ? "rejected" : "neutral";

    const majorityPercent = Math.max(upvotes, downvotes) * 100 / (upvotes + downvotes);
    const percentText = " (" + Math.round(majorityPercent) + "% " + ((result > 0) ? "+" : "-") + ")";

    return <span className={"inline-voting"}>
      <FastButton onClick={this.upvote} className='approve'>
        <MessageText content={':thumbsup:'} onlyEmoji /> {upvotes}
      </FastButton>
      <FastButton onClick={this.downvote} className='disapprove'>
        <MessageText content={':thumbsdown:'} onlyEmoji /> {downvotes}
      </FastButton>
      <span className={resultClass}> {result}</span>
      {result != 0 && <small>{percentText}</small>}
    </span>
  }

})
