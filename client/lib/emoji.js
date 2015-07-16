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
index['euphoria!'] = 'euphoric'
index['chromakode'] = 'chromakode'
index['pewpewpew'] = 'pewpewpew'
index['leck'] = 'leck'
index['dealwithit'] = 'dealwithit'
index['spider'] = 'spider'
index['indigo_heart'] = 'indigo_heart'
index['orange_heart'] = 'orange_heart'

module.exports.names = _.invert(index)

module.exports.codes = _.uniq(_.values(index))
