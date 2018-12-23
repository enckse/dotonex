object = network:define("2", "test1")
object.macs = {VALID_MAC, "aabbccddeeff"}
object:assigned(DEV)
object:assigned(PROD)

object = network:define("2", "test1")
object.macs = {"ffddeeffddee", "aabbaabbaabb"}
object:mabed(PROD)
