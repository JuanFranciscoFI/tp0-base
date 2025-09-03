from common.utils import Bet

STATUS_OK = 0x00
STATUS_FAIL = 0x01

class Protocol:
    def __init__(self, sock, agency_id: int = 0):
        self.sock = sock
        self.agency_id = agency_id

    #evitar short-reads
    def _read_all(self, nbytes: int) -> bytes:
        buf = b''
        while len(buf) < nbytes:
            chunk = self.sock.recv(nbytes - len(buf))
            if not chunk:
                return None
            buf += chunk
        return buf

    def _recv_u16(self) -> int:
        return int.from_bytes(self._read_all(2), "big", signed=False)

    def _recv_u32(self) -> int:
        return int.from_bytes(self._read_all(4), "big", signed=False)

    def _recv_string_u16(self) -> str:
        length = self._recv_u16()
        data = self._read_all(length)
        if not data:
            return None
        return data.decode("utf-8")

    def recv_bet(self) -> Bet:
        first_name = self._recv_string_u16()
        last_name = self._recv_string_u16()
        document = self._recv_u32()
        birthdate = self._recv_string_u16()
        birthdate = birthdate.strip().strip('"').strip("'")
        number = self._recv_u32()

        return Bet(
            agency=self.agency_id,
            first_name=first_name,
            last_name=last_name,
            document=document,
            birthdate=birthdate,
            number=number,
        )

    def send_response(self, ok: bool = True) -> None:
        value = STATUS_OK if ok else STATUS_FAIL
        self.sock.sendall(bytes([value]))
