import React from 'react'

import chat from '../stores/chat'
import ui from '../stores/ui'
import hueHash from '../hueHash'
import FastButton from './FastButton'


export default function PMNotice({ pmId, nick, kind }) {
  const bgColor = 'hsl(' + hueHash.hue(nick) + ', 67%, 85%)'
  const textLightColor = 'hsl(' + hueHash.hue(nick) + ', 28%, 28%)'
  const textColor = 'hsl(' + hueHash.hue(nick) + ', 28%, 43%)'
  return (
    <div className="notice light pm" style={{backgroundColor: bgColor}}>
      <div className="content">
        <span className="title" style={{color: textLightColor}}>{kind === 'from' ? `${nick} invites you to a private conversation` : `you invited ${nick} to a private conversation`}</span>
        <div className="actions">
          <FastButton onClick={() => ui.openPMWindow(pmId)} style={{color: textColor}}>join room</FastButton>
        </div>
      </div>
      <FastButton className="close" onClick={() => chat.dismissPMNotice(pmId)} />
    </div>
  )
}

PMNotice.propTypes = {
  pmId: React.PropTypes.string.isRequired,
  nick: React.PropTypes.string.isRequired,
  kind: React.PropTypes.string.isRequired,
}
