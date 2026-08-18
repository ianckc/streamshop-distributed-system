import logging
import sys

import uvicorn

from analytics_api.app import create_app
from analytics_api.config import Config
from analytics_api.telemetry import init_telemetry


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(message)s",
        stream=sys.stdout,
    )
    config = Config.from_env()
    init_telemetry(config.service_name)
    app = create_app(config)
    uvicorn.run(app, host="0.0.0.0", port=config.port, log_config=None)


if __name__ == "__main__":
    main()
