import socket
import logging
import signal
import threading
from collections import defaultdict

from common.protocol import Protocol, MSG_HELLO, MSG_BATCH, MSG_DONE
from common.utils import store_bets, load_bets, has_won


class Server:
    def __init__(self, port, listen_backlog):
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self._server_socket.bind(('', port))
        self._server_socket.listen(listen_backlog)
        self._shutdown = False

        self._expected_agencies = None
        self._bets_lock = threading.Lock()
        self._barrier = None
        self._winners_by_agency = {}

        self._workers = []

    def __graceful_shutdown(self, signum, frame):
        logging.info("action: graceful_shutdown | result: in_progress")
        self._shutdown = True
        try:
            self._server_socket.close()
        except Exception:
            pass
        b = self._barrier
        if b is not None:
            try:
                b.abort()
            except Exception:
                pass
        logging.info("action: graceful_shutdown | result: success")

    def run(self, expected_agencies):
        self._expected_agencies = expected_agencies
        self._barrier = threading.Barrier(
            parties=expected_agencies,
            action=self.__compute_winners
        )

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

                t = threading.Thread(
                    target=self.__handle_client_until_done,
                    args=(proto,),
                    name=f"client-{addr[0]}:{addr[1]}",
                    daemon=True
                )
                t.start()
                self._workers.append(t)
        finally:
            try:
                self._server_socket.close()
            except Exception:
                pass
            for t in self._workers:
                t.join(timeout=1.0)

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
                    with self._bets_lock:
                        store_bets(bets)
                    proto.send_ack(True)
                    logging.info(f"action: apuesta_recibida | result: success | cantidad: {len(bets)}")

                elif mtype == MSG_DONE:
                    if agency_id is None:
                        proto.send_ack(False)
                        continue
                    proto.send_ack(True)

                    try:
                        self._barrier.wait()
                    except threading.BrokenBarrierError:
                        logging.error("action: barrier_wait | result: fail | error: BrokenBarrier")
                        try:
                            proto.close()
                        except Exception:
                            pass
                        return

                    dnis = self._winners_by_agency.get(agency_id, [])
                    try:
                        proto.send_winners(dnis)
                        logging.info(f"action: send_lottery_results | result: success | agency: {agency_id} | winners: {len(dnis)}")
                    except Exception as e:
                        logging.error(f"action: send_lottery_results | result: fail | agency: {agency_id} | error: {e}")
                    finally:
                        try:
                            proto.close()
                        except Exception:
                            pass
                    return

                else:
                    try:
                        proto.send_ack(False)
                    except Exception:
                        pass
                    proto.close()
                    return

        except ConnectionError:
            try:
                proto.close()
            except Exception:
                pass
        except Exception as e:
            logging.error(f"action: server_error | error: {e}")
            try:
                proto.send_ack(False)
            except Exception:
                pass
            try:
                proto.close()
            except Exception:
                pass

    def __compute_winners(self):
        winners_by_agency = defaultdict(list)
        with self._bets_lock:
            for bet in load_bets():
                if has_won(bet):
                    try:
                        winners_by_agency[bet.agency].append(int(bet.document))
                    except Exception:
                        pass
        self._winners_by_agency = winners_by_agency
        logging.info("action: sorteo | result: success")
