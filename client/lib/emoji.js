var _ = require('lodash')


module.exports = {}

var index = module.exports.index = _.clone(require('emoji-annotation-to-unicode'))

index['+1'] = 'plusone'

module.exports.names = _.invert(index)

module.exports.codes = _.uniq(_.values(index))
