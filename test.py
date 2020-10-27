from fluent.sender import FluentSender

logger = FluentSender("app", host="localhost", port=24224)

logger.emit("follow", {"from": "userA", "to": "userB"})
logger.emit("bof", dict(beuha="aussi", age=42))

logger.close()
