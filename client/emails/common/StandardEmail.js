import React from 'react'

import { Email } from '../email'


export default function StandardEmail({children}) {
  return (
    <Email title="{{.Subject}}" bgcolor="#f0f0f0" cellSpacing={30}>
      {children}
    </Email>
  )
}

StandardEmail.propTypes = {
  children: React.PropTypes.node,
}
