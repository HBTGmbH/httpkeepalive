package gracefulshutdown;
import org.apache.hc.client5.http.classic.HttpClient;
import org.apache.hc.client5.http.impl.classic.HttpClients;
import org.apache.hc.client5.http.impl.io.PoolingHttpClientConnectionManagerBuilder;
import org.apache.hc.client5.http.io.HttpClientConnectionManager;
import org.apache.hc.core5.util.TimeValue;
import org.springframework.boot.restclient.RestTemplateBuilder;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;
import org.springframework.http.client.HttpComponentsClientHttpRequestFactory;
@Configuration
public class Config {
    @Bean
    public RestTemplateBuilder restTemplateBuilder(HttpComponentsClientHttpRequestFactory requestFactory) {
        return new RestTemplateBuilder().requestFactory(() -> requestFactory);
    }
    @Bean
    public HttpComponentsClientHttpRequestFactory requestFactory(HttpClient httpClient) {
        return new HttpComponentsClientHttpRequestFactory(httpClient);
    }
    @Bean
    public HttpClientConnectionManager httpClientConnectionManager() {
        return PoolingHttpClientConnectionManagerBuilder.create()
                .setMaxConnPerRoute(100)
                .setMaxConnTotal(200)
                .build();
    }
    @Bean
    public HttpClient closeableHttpClient(HttpClientConnectionManager cm) {
        return HttpClients.custom()
                .setConnectionManager(cm)
                .evictExpiredConnections()
                .evictIdleConnections(TimeValue.ofSeconds(5))
                .build();
    }
}

