import socket
import logging
import signal
from collections import defaultdict

from common.protocol import Protocol, MSG_HELLO, MSG_BATCH, MSG_DONE
from common.utils import store_bets, load_bets, has_won
class Server:
    def __init__(self, port, listen_backlog):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._shutdown = False

        self._agents_waiting = {}
        self._draw_done = False

    def __graceful_shutdown(self, signum, frame):
        logging.info("action: graceful_shutdown | result: in_progress")
        self._shutdown = True
        try:
            self._server_socket.close()
        finally:
            for proto in self._agents_waiting.values():
                proto.close()
            logging.info("action: graceful_shutdown | result: success")

    def run(self, expected_agencies):
        signal.signal(signal.SIGINT, self.__graceful_shutdown)
        signal.signal(signal.SIGTERM, self.__graceful_shutdown)

        try:
            while not self._shutdown:
                logging.info('action: accept_connections | result: in_progress')
                try:
                    c, addr = self._server_socket.accept()
                except OSError:
                    if self._shutdown:
                        break
                    continue

                logging.info(f'action: accept_connections | result: success | ip: {addr[0]}')
                proto = Protocol(c)

                self.__handle_client_until_done(proto)

                if not self._draw_done and len(self._agents_waiting) == expected_agencies:
                    self.broadcast_results()
        finally:
            self._server_socket.close()
            
    def __handle_client_until_done(self, proto: Protocol):
        agency_id = None
        try:
            while True:
                mtype = proto.recv_msg_type()
                logging.info(f"action: msg_type | result: success | type: {mtype}")

                if mtype == MSG_HELLO:
                    agency_id = proto.recv_u16()
                    proto.send_ack(True)

                elif mtype == MSG_BATCH:
                    if agency_id is None:
                        proto.send_ack(False)
                        continue
                    bets = proto.recv_batch(agency_id)
                    store_bets(bets)
                    proto.send_ack(True)
                    logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")

                elif mtype == MSG_DONE:
                    if agency_id is None:
                        proto.send_ack(False)
                        continue
                    proto.send_ack(True)
                    self._agents_waiting[agency_id] = proto
                    return

                else:
                    proto.send_ack(False)
                    proto.close()
                    return
        except ConnectionError:
            proto.close()
        except Exception as e:
            logging.error(f"action: server_error | error: {e}")
            proto.send_ack(False)
            proto.close()

    def lottery(self):
        winners_by_agency = defaultdict(list)
        for bet in load_bets():
            if has_won(bet):
                winners_by_agency[bet.agency].append(int(bet.document))
        return winners_by_agency

    def broadcast_results(self):
        winners_by_agency = self.lottery()

        for ag_id, proto in list(self._agents_waiting.items()):
            dnis = winners_by_agency.get(ag_id, [])
            try:
                proto.send_winners(dnis)
                logging.info(f"action: send_lottery_results | result: success | agency: {ag_id} | winners: {len(dnis)}")
            except Exception as e:
                logging.error(f"action: send_lottery_results | result: fail | agency: {ag_id} | error: {e}")
            finally:
                proto.close()

        self._agents_waiting.clear()
        self._draw_done = True
        logging.info("action: sorteo | result: success")
