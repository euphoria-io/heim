var _ = require('lodash')


module.exports = {}

var index = module.exports.index = _.clone(require('emoji-annotation-to-unicode'))

// (don't nag about object notation)
// jshint -W069
index['+1'] = 'plusone'
index['bronze'] = 'bronze'
index['bronze!?'] = 'bronze2'
index['bronze?!'] = 'bronze2'
index['euphoria'] = 'euphoria'
index['chromakode'] = 'chromakode'

module.exports.names = _.invert(index)

module.exports.codes = _.uniq(_.values(index))
