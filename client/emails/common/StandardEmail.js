import React from 'react'

import { Email } from 'react-html-email'


export default function StandardEmail({children}) {
  return (
    <Email title="{{.Subject}}" bgcolor="#f0f0f0" cellSpacing={10} style={{paddingTop: '20px', paddingBottom: '20px'}}>
      {children}
    </Email>
  )
}

StandardEmail.propTypes = {
  children: React.PropTypes.node,
}
