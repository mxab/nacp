import com.sun.net.httpserver.*;
import java.io.*;
import java.net.InetSocketAddress;
import static java.lang.System.getenv;
import static java.lang.Integer.parseInt;
import java.util.logging.*;

public class MockHooks {
    private static final Logger logger = Logger.getLogger(MockHooks.class.getName());

    public static void main(String[] args) throws IOException {
        logger.info("Starting MockHooks server");
        var port = parseInt(getenv("PORT"));
        var server = HttpServer.create(new InetSocketAddress(port), 0);

        server.createContext("/validate", exchange -> {
            logger.info("Received validation request");
            writeJson(exchange, """
            {
                "warnings": [],
                "errors": []
            }
            """);
        });
        server.createContext("/mutate", exchange -> {
            logger.info("Received mutation request");
            writeJson(exchange, """
            {
                "warnings": [],
                "errors": [],
                "patch" : []
            }
            """);
        });
        logger.info("Mock hooks server started on port " + port);
        server.start();

    }

    private static void writeJson(HttpExchange exchange, String response) throws IOException {
        exchange.sendResponseHeaders(200, response.length());
        var out = exchange.getResponseBody();
        out.write(response.getBytes());
        out.close();
    }
}
