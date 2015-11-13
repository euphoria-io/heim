import React from 'react'

import { Item, Box } from '../email'


export default function BodyBox({children}) {
  return (
    <Item style={{paddingTop: '20px'}}>
      <Box cellSpacing={20} width="100%" bgcolor="white" style={{borderBottom: '3px solid #ccc'}}>
        {children}
      </Box>
    </Item>
  )
}

BodyBox.propTypes = {
  children: React.PropTypes.node,
}
