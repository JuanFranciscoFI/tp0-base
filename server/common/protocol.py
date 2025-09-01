import struct
from common.utils import Bet

STATUS_OK  = 0x00
STATUS_FAIL = 0x01

class Protocol:
    def __init__(self, sock, agency_id: int = 0):
        self.sock = sock
        self.agency_id = agency_id

    def _read_all(self, nbytes: int) -> bytes:
        buf = b''
        while len(buf) < nbytes:
            chunk = self.sock.recv(nbytes - len(buf))
            if not chunk:
                raise ConnectionError("unexpected EOF")
            buf += chunk
        return buf

    def _recv_u16(self) -> int:
        return struct.unpack(">H", self._read_all(2))[0]

    def _recv_u32(self) -> int:
        return struct.unpack(">I", self._read_all(4))[0]

    def _recv_string_u16(self) -> str:
        length = self._recv_u16()
        if length == 0:
            return ""
        data = self._read_all(length)
        return data.decode("utf-8")

    def recv_bet(self) -> Bet:
        first_name = self._recv_string_u16()
        last_name  = self._recv_string_u16()
        document   = str(self._recv_u32())
        birthdate  = self._recv_string_u16().strip().strip('"').strip("'")
        number     = self._recv_u32()

        return Bet(
            agency=self.agency_id,
            first_name=first_name,
            last_name=last_name,
            document=document,
            birthdate=birthdate,
            number=number,
        )

    def recv_batch(self) -> list[Bet]:
        count = self._recv_u16()
        if count == 0:
            return []
        bets = []
        for _ in range(count):
            bets.append(self.recv_bet())
        return bets

    def send_response(self, ok: bool = True) -> None:
        value = STATUS_OK if ok else STATUS_FAIL
        self.sock.sendall(bytes([value]))
