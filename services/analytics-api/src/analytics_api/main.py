import logging
import sys

import uvicorn

from analytics_api.app import create_app
from analytics_api.config import Config


def main() -> None:
    logging.basicConfig(
        level=logging.INFO,
        format="%(message)s",
        stream=sys.stdout,
    )
    config = Config.from_env()
    app = create_app(config)
    uvicorn.run(app, host="0.0.0.0", port=config.port, log_config=None)


if __name__ == "__main__":
    main()
