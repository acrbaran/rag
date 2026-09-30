class RethraLite < Formula
  desc "Knowledge base management system — single-binary Lite edition"
  homepage "https://github.com/acrbaran/rag"
  version "0.3.6-test"
  license "Apache-2.0"

  on_macos do
    on_arm do
      url "https://github.com/acrbaran/rag/releases/download/v#{version}/Rethra-lite_v#{version}_darwin_arm64.tar.gz"
      sha256 "1da2d4eef99e5cf8aa7a58501baa059e9e20482e1bd65a36a82321a89926c104"
    end
    on_intel do
      url "https://github.com/acrbaran/rag/releases/download/v#{version}/Rethra-lite_v#{version}_darwin_amd64.tar.gz"
      sha256 "c187e16ac7671a615f012c82ebd89786e11fcf67cccc773eff175e4bdf7c9c06"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/acrbaran/rag/releases/download/v#{version}/Rethra-lite_v#{version}_linux_arm64.tar.gz"
      sha256 "bc4e184da005b60d1e8c037a61c58e643ebdc9bf14470fae6cd6227f52f02f1c"
    end
    on_intel do
      url "https://github.com/acrbaran/rag/releases/download/v#{version}/Rethra-lite_v#{version}_linux_amd64.tar.gz"
      sha256 "cb34c50fb5b05555fca16084ffc7710524ff78badb3b1b82474eb89d21545d6e"
    end
  end

  def install
    libexec.install "Rethra-lite"
    pkgshare.install "web" if File.directory?("web")
    pkgshare.install "config" if File.directory?("config")
    pkgshare.install ".env.lite.example"
    doc.install "README.md"
    pkgshare.install "migrations" if File.directory?("migrations")

    (bin/"rethra-lite").write <<~SH
      #!/bin/bash
      CONFIG_DIR="${RETHRA_CONFIG_DIR:-${XDG_CONFIG_HOME:-$HOME/.config}/rethra}"
      DATA_DIR="${RETHRA_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/rethra}"

      mkdir -p "$DATA_DIR/files" "$CONFIG_DIR/config" 2>/dev/null

      if [ ! -f "$CONFIG_DIR/config/config.yaml" ]; then
        cp -r "#{pkgshare}/config/" "$CONFIG_DIR/config/"
      fi

      if [ ! -d "$CONFIG_DIR/migrations" ] && [ -d "#{pkgshare}/migrations" ]; then
        ln -sf "#{pkgshare}/migrations" "$CONFIG_DIR/migrations"
      fi

      if [ ! -f "$CONFIG_DIR/.env.lite" ]; then
        cp "#{pkgshare}/.env.lite.example" "$CONFIG_DIR/.env.lite"
        sed -i '' "s|DB_PATH=.*|DB_PATH=$DATA_DIR/rethra.db|" "$CONFIG_DIR/.env.lite"
        sed -i '' "s|LOCAL_STORAGE_BASE_DIR=.*|LOCAL_STORAGE_BASE_DIR=$DATA_DIR/files|" "$CONFIG_DIR/.env.lite"
        rm -f "$CONFIG_DIR/.env.lite-e"
        echo ""
        echo "Yapılandırma dosyası oluşturuldu: $CONFIG_DIR/.env.lite"
        echo "Lütfen gerektiğinde düzenleyin (örneğin LLM adresi, güvenlik anahtarı vb.)."
        echo ""
      fi

      set -a
      source "$CONFIG_DIR/.env.lite"
      set +a

      export DB_PATH="${DB_PATH:-$DATA_DIR/rethra.db}"
      export LOCAL_STORAGE_BASE_DIR="${LOCAL_STORAGE_BASE_DIR:-$DATA_DIR/files}"
      export RETHRA_WEB_DIR="${RETHRA_WEB_DIR:-#{pkgshare}/web}"

      cd "$CONFIG_DIR"
      exec "#{libexec}/Rethra-lite" "$@"
    SH
  end

  def post_install
    (var/"rethra").mkpath
    (var/"log").mkpath
  end

  service do
    run [bin/"rethra-lite"]
    keep_alive true
    working_dir var/"rethra"
    log_path var/"log/rethra-lite.log"
    error_log_path var/"log/rethra-lite.log"
  end

  def caveats
    <<~EOS
      Ön planda çalıştırma:
        rethra-lite

      Arka plan servisi (önerilen):
        brew services start rethra-lite   # Başlat ve açılışta otomatik başlat
        brew services stop rethra-lite    # Durdur
        brew services restart rethra-lite # Yeniden başlat
        brew services info rethra-lite    # Durumu görüntüle

      Günlükler:
        #{var}/log/rethra-lite.log

      İlk çalıştırmada yapılandırma dosyası otomatik oluşturulur:
        ~/.config/rethra/.env.lite

      Veriler şurada saklanır:
        ~/.local/share/rethra/

      Yapılandırmayı değiştirmek için (LLM servis adresi, güvenlik anahtarları vb.):
        $EDITOR ~/.config/rethra/.env.lite
        brew services restart rethra-lite
    EOS
  end

  test do
    assert_predicate bin/"rethra-lite", :executable?
  end
end
