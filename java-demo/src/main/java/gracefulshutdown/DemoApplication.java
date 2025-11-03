package gracefulshutdown;
import jakarta.servlet.http.HttpServletRequest;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.boot.restclient.RestTemplateBuilder;
import org.springframework.http.HttpMethod;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.util.UriComponentsBuilder;
import java.net.URI;
@SpringBootApplication
@RestController
public class DemoApplication {
    private final RestTemplate restTemplate;
    public DemoApplication(RestTemplateBuilder restTemplateBuilder) {
        this.restTemplate = restTemplateBuilder.build();
    }
    private String proxy(String serviceName, HttpServletRequest request) {
        HttpMethod method = HttpMethod.valueOf(request.getMethod());
        URI targetUri = UriComponentsBuilder.fromUriString("http://" + serviceName)
                .path(request.getRequestURI().substring(("/" + serviceName).length()))
                .query(request.getQueryString())
                .build(true)
                .toUri();
        return restTemplate.exchange(targetUri, method, null, String.class).getBody();
    }
    @RequestMapping(path = "/envoy/**", method = { RequestMethod.GET, RequestMethod.POST })
    public String envoy(HttpServletRequest request) {
        return proxy("envoy", request);
    }
    @RequestMapping(path = "/nginx/**", method = { RequestMethod.GET, RequestMethod.POST })
    public String nginx(HttpServletRequest request) {
        return proxy("nginx", request);
    }
    @RequestMapping(path = "/varnish/**", method = { RequestMethod.GET, RequestMethod.POST })
    public String varnish(HttpServletRequest request) {
        return proxy("varnish", request);
    }
    @RequestMapping(path = "/go-demo/**", method = { RequestMethod.GET, RequestMethod.POST })
    public String goDemo(HttpServletRequest request) {
        return proxy("go-demo", request);
    }
    @RequestMapping(path = "/node-demo/**", method = { RequestMethod.GET, RequestMethod.POST })
    public String nodeDemo(HttpServletRequest request) {
        return proxy("node-demo", request);
    }
    @RequestMapping(path = "/sleep",
            method = { RequestMethod.GET, RequestMethod.POST })
    public String sleep(@RequestParam(defaultValue = "50") long min,
                         @RequestParam(defaultValue = "1000") long max) throws Exception {
        min = Math.max(0, min);
        max = Math.min(30000, Math.max(min, max));
        long sleepTime = min + (long) (Math.random() * (max - min));
        Thread.sleep(sleepTime);
        return "Slept for: " + sleepTime + " ms.\n";
    }
    public static void main(String[] args) {
        SpringApplication.run(DemoApplication.class, args);
    }
}

