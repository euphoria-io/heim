import fs from 'fs'
import React from 'react'

import { MainPage, PolicyNav, Markdown } from '../common'


module.exports = (
  <MainPage title="euphoria: copyright policy" nav={<PolicyNav selected="dmca" />}>
    <Markdown className="policy" content={fs.readFileSync(__dirname + '/dmca.md', 'utf8')} />
  </MainPage>
)
