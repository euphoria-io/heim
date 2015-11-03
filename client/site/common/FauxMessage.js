import React from 'react'

import MessageText from '../../lib/ui/message-text'
import FauxNick from './FauxNick'


export default function FauxMessage(props) {
  return (
    <div className="faux-message">
      <div className="line">
        <FauxNick nick={props.sender} />
        <div className="content">
          <MessageText className="message" content={props.message} />
          {props.embed && <div className="embed">
            <div className="wrapper">
              <img className="embed" src={props.embed} alt="" />
            </div>
          </div>}
        </div>
      </div>
      {props.children}
    </div>
  )
}

FauxMessage.propTypes = {
  sender: React.PropTypes.string,
  message: React.PropTypes.string,
  embed: React.PropTypes.string,
  children: React.PropTypes.node,
}
