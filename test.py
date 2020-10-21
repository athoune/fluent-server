from fluent import sender

logger = sender.FluentSender('app', host='localhost', port=24224)

logger.emit('follow', {'from': 'userA', 'to': 'userB'})
logger.emit('bof', dict(beuha="aussi", age=42))
