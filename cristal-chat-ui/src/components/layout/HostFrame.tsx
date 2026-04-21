import React from 'react';
import BrowserChrome from './BrowserChrome';
import Header from './Header';
import YellowDivider from './YellowDivider';
import Footer from './Footer';

interface HostFrameProps {
  children: React.ReactNode;
}

/**
 * Moldura hospedeira - simula integração em site institucional
 * Layout completo com chrome do navegador, header, conteúdo e footer
 */
const HostFrame: React.FC<HostFrameProps> = ({ children }) => {
  return (
    <div className="h-screen overflow-hidden bg-gray-200 p-0 sm:p-8">
      <div className="h-full max-w-[720px] mx-auto bg-white shadow-2xl rounded-none sm:rounded-lg overflow-hidden flex flex-col">
        <BrowserChrome />
        <Header />
        <YellowDivider />

        {/* Main content */}
        <main className="flex-1 flex flex-col overflow-hidden">
          {children}
        </main>

        <Footer />
      </div>
    </div>
  );
};

export default HostFrame;
