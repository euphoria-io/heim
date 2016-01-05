import React from 'react'

import { Item, Box } from 'react-html-email'


export default function BodyBox({children}) {
  return (
    <Item style={{paddingTop: '12px'}}>
      <Box cellPadding={20} width="100%" bgcolor="white" style={{borderBottom: '3px solid #ccc'}}>
        {children}
      </Box>
    </Item>
  )
}

BodyBox.propTypes = {
  children: React.PropTypes.node,
}
