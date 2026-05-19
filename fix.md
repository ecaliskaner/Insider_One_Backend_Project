services/league_impl.go (line 149) içindeki PlayAll() deadlock yapar.

PlayAll() stateMu.Lock() alıyor, sonra aynı dosyada (line 162) PlayNextWeek() çağırıyor. PlayNextWeek() de aynı mutex’i tekrar almaya çalışıyor. Go sync.Mutex reentrant değildir, yani /api/v1/league/play-all takılır. Bu case’in ekstra maddelerinden biri olduğu için çok önemli.



go test ./... başarısız.

services/match_engine_test.go (line 67) ve services/match_engine_test.go (line 68) assertion’ları beklenen skorlarla gerçek skorları tutturamıyor. CI da go test -v ./... çalıştırdığı için GitHub Actions kırılır. Yarışmalı bir case’te kırmızı CI çok kötü görünür.



README ve loglarda encoding bozulmuş.

README’de âš½, ğŸ gibi mojibake karakterler var. Kodda da log mesajları bozuk görünüyor. Teknik olarak sistemi bozmayabilir ama ilk izlenimde “özen” puanı düşürür.



Event-driven yapı biraz vitrin gibi kalmış.

services/listeners.go (line 8) listener morale/fatigue güncelliyor, ama aynı zamanda PlayNextWeek() sonunda standings yeniden hesaplanıyor. Ayrıca event publish async olduğu için takım metriklerinin ne zaman güncellendiği deterministik değil. Bu mimariyi ya sadeleştir ya da transaction/event consistency açısından netleştir.



Edit match metrikleri tam güvenilir değil.

services/league_impl.go (line 188) maç sonucu editlenince standings yeniden hesaplanıyor ama morale/fatigue eski oynanmış maçların tümünden deterministik yeniden üretilmiyor; sadece editlenen maç için tekrar değişiklik uygulanıyor. Bu zamanla şişen/yanlış metriklere yol açabilir.



Senior Backend Gözüyle Değerlendirme

Fikir olarak proje güçlü: interface-based design, repository katmanı, migrations, Swagger, Docker, CI, Monte Carlo tahmin, rollback, rate limit, graceful shutdown gibi şeyler staj case’i için normal beklentinin üstünde. “Ben sadece endpoint yazmadım, backend sistemi düşündüm” mesajı veriyor.

Ama şu haliyle biraz fazla özellik eklenmiş, bazıları tam oturmamış. Senior bakışla en büyük sorun “gösterişli ama kırılgan” algısı. Özellikle deadlock ve failing test, mimari iddiaların güvenilirliğini zedeliyor. Önce küçük ama sağlam bir çekirdek, sonra ekstra özellikler daha iyi görünür.

Öncelikli İyileştirme Sırası

1) PlayAll() deadlock’unu düzelt. Ortak internal playNextWeekLocked() gibi mutex almayan private fonksiyon çıkar.
2) Testleri düzelt ve go test ./... yeşil olmadan teslim etme.
3) Edit/rollback sonrası standings ve team metrics hesaplamasını tek deterministik “rebuild state from matches” fonksiyonuna bağla.
4) API response formatını standartlaştır: success, data, error, meta.
5) context.Context repository/service katmanına geçir. Bu senior backend artısıdır.
6) Integration test ekle: reset → 4 hafta oyna → championship probabilities → edit → rollback → play-all.
7) Postman collection veya Bruno collection ekle. Değerlendiren kişi tek tıkla deneyebilsin.
8) README’ye “Case Requirements Mapping” tablosu koy. Değerlendirici aradığı maddeyi hemen görsün.
9) Canlı deploy linki varsa çok öne geçirir. Render/Fly.io/Railway gibi basit bir deployment yeterli.
