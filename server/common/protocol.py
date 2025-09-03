from common.utils import Bet

STATUS_OK   = 0x00
STATUS_FAIL = 0x01

MSG_HELLO        = 0x01
MSG_BATCH        = 0x02
MSG_DONE         = 0x03 

class Protocol:
    def __init__(self, sock):
        self.sock = sock

    def close(self):
        try:
            self.sock.close()
        except Exception:
            pass

    def _read_all(self, nbytes: int) -> bytes:
        buf = b''
        while len(buf) < nbytes:
            chunk = self.sock.recv(nbytes - len(buf))
            if not chunk:
                raise ConnectionError("unexpected EOF")
            buf += chunk
        return buf

    def recv_msg_type(self) -> int:
        return self._read_all(1)[0]

    def recv_u16(self) -> int:
        return int.from_bytes(self._read_all(2), "big", signed=False)

    def recv_u32(self) -> int:
        return int.from_bytes(self._read_all(4), "big", signed=False)

    def recv_string_u16(self) -> str:
        length = self.recv_u16()
        if length == 0:
            return ""
        data = self._read_all(length)
        return data.decode("utf-8")

    def recv_batch(self, agency_id: int) -> list[Bet]:
        count = self.recv_u16()
        bets = []
        for _ in range(count):
            first_name = self.recv_string_u16()
            last_name  = self.recv_string_u16()
            document   = str(self.recv_u32())
            birthdate  = self.recv_string_u16().strip().strip('"').strip("'")
            number     = self.recv_u32()
            bets.append(Bet(
                agency=agency_id,
                first_name=first_name,
                last_name=last_name,
                document=document,
                birthdate=birthdate,
                number=number,
            ))
        return bets

    def send_ack(self, ok: bool) -> None:
        self.sock.sendall(bytes([STATUS_OK if ok else STATUS_FAIL]))

    def send_winners(self, dnis: list[int]) -> None:
        parts = [bytes([STATUS_OK])]

        n = len(dnis)
        if not (0 <= n <= 0xFFFF):
            raise ValueError("too many winners for u16 length")
        parts.append(n.to_bytes(2, "big", signed=False))

        for dni in dnis:
            d = int(dni)
            if not (0 <= d <= 0xFFFFFFFF):
                raise ValueError(f"dni out of range for u32: {dni}")
            parts.append(d.to_bytes(4, "big", signed=False))

        self.sock.sendall(b"".join(parts))

