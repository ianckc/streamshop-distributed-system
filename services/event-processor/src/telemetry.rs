use opentelemetry::propagation::Extractor;
use opentelemetry::Context;
use opentelemetry_sdk::propagation::TraceContextPropagator;
use rdkafka::message::{BorrowedHeaders, Headers};

pub fn init_propagation() {
    opentelemetry::global::set_text_map_propagator(TraceContextPropagator::new());
}

pub fn extract_context(headers: Option<&BorrowedHeaders>) -> Context {
    let extractor = KafkaHeaderExtractor { headers };
    opentelemetry::global::get_text_map_propagator(|propagator| propagator.extract(&extractor))
}

struct KafkaHeaderExtractor<'a> {
    headers: Option<&'a BorrowedHeaders>,
}

impl Extractor for KafkaHeaderExtractor<'_> {
    fn get(&self, key: &str) -> Option<&str> {
        let headers = self.headers?;
        for i in 0..headers.count() {
            let header = headers.get(i);
            if header.key == key {
                return header.value.and_then(|v| std::str::from_utf8(v).ok());
            }
        }
        None
    }

    fn keys(&self) -> Vec<&str> {
        let Some(headers) = self.headers else {
            return Vec::new();
        };
        (0..headers.count()).map(|i| headers.get(i).key).collect()
    }
}
