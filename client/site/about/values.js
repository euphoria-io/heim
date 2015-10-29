import fs from 'fs'
import React from 'react'

import { MainPage, PolicyNav, Markdown } from '../common'


module.exports = (
  <MainPage title="euphoria: values" nav={<PolicyNav selected="values" />}>
    <Markdown className="policy" content={fs.readFileSync(__dirname + '/values.md', 'utf8')} />
  </MainPage>
)
